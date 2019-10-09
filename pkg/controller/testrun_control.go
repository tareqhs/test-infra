// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/go-logr/logr"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconcile handles various testrun events like crete, update and delete.
func (r *TestrunReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := &reconcileContext{
		ctx: context.Background(),
		tr:  &tmv1beta1.Testrun{},
	}
	defer ctx.ctx.Done()
	log := r.Logger.WithValues("testrun", request.NamespacedName)

	log.V(3).Info("start reconcile")

	err := r.Get(ctx.ctx, request.NamespacedName, ctx.tr)
	if err != nil {
		log.Error(err, "unable to find testrun")
		if errors.IsNotFound(err) {
			return reconcile.Result{Requeue: false}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if ctx.tr.DeletionTimestamp != nil {
		log.Info("deletion caused by testrun")
		return r.deleteTestrun(ctx)
	}

	///////////////
	// RECONCILE //
	///////////////

	ctx.wf = &argov1.Workflow{}
	err = r.Get(ctx.ctx, types.NamespacedName{Name: testmachinery.GetWorkflowName(ctx.tr), Namespace: ctx.tr.Namespace}, ctx.wf)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to get workflow", "workflow", testmachinery.GetWorkflowName(ctx.tr), "namespace", ctx.tr.Namespace)
			return reconcile.Result{}, err
		}
		if ctx.tr.Status.CompletionTime != nil {
			log.Error(err, "unable to get workflow but testrun is already finished", "workflow", testmachinery.GetWorkflowName(ctx.tr), "namespace", ctx.tr.Namespace)
			return reconcile.Result{}, err
		}

		if res, err := r.createWorkflow(ctx, log); err != nil {
			return res, err
		}
	}

	if ctx.tr.Status.CompletionTime != nil {
		if ctx.wf.DeletionTimestamp != nil {
			log.V(2).Info("Deletion: cause workflow")
			return r.deleteTestrun(ctx)
		}
		return reconcile.Result{}, err
	}

	if err := r.handleActions(ctx); err != nil {
		return reconcile.Result{}, err
	}

	return r.updateStatus(ctx)

}

func (r *TestrunReconciler) createWorkflow(ctx *reconcileContext, log logr.Logger) (reconcile.Result, error) {
	log.V(5).Info("generate workflow")
	var err error
	ctx.wf, err = r.generateWorkflow(ctx.ctx, ctx.tr)
	if err != nil {
		log.Error(err, "unable to setup workflow")
		return reconcile.Result{}, err
	}
	log.Info("creating workflow", "workflow", ctx.wf.Name, "namespace", ctx.wf.Namespace)
	err = r.Create(ctx.ctx, ctx.wf)
	if err != nil {
		r.Logger.Error(err, "unable to create workflow", "workflow", ctx.wf.Name, "namespace", ctx.wf.Namespace)
		return reconcile.Result{Requeue: true}, err
	}

	ctx.tr.Status.Workflow = ctx.wf.Name
	ctx.tr.Status.Phase = tmv1beta1.PhaseStatusRunning

	// add finalizers for testrun
	trFinalizers := sets.NewString(ctx.tr.Finalizers...)
	if !trFinalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
		trFinalizers.Insert(tmv1beta1.SchemeGroupVersion.Group)
	}
	if !trFinalizers.Has(metav1.FinalizerDeleteDependents) {
		trFinalizers.Insert(metav1.FinalizerDeleteDependents)
	}
	ctx.tr.Finalizers = trFinalizers.UnsortedList()

	ctx.updated = true
	return reconcile.Result{}, nil
}

func (r *TestrunReconciler) generateWorkflow(ctx context.Context, testrunDef *tmv1beta1.Testrun) (*argov1.Workflow, error) {
	tr, err := testrun.New(r.Logger.WithValues("testrun", types.NamespacedName{Name: testrunDef.Name, Namespace: testrunDef.Namespace}), testrunDef)
	if err != nil {
		return nil, fmt.Errorf("error parsing testrun: %s", err.Error())
	}

	wf, err := tr.GetWorkflow(testmachinery.GetWorkflowName(testrunDef), testrunDef.Namespace, r.getImagePullSecrets(ctx))
	if err != nil {
		return nil, err
	}

	if err := controllerutil.SetControllerReference(testrunDef, wf, r.scheme); err != nil {
		return nil, err
	}

	wfFinalizers := sets.NewString(wf.Finalizers...)
	if !wfFinalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
		wfFinalizers.Insert(tmv1beta1.SchemeGroupVersion.Group)
		wf.Finalizers = wfFinalizers.UnsortedList()
	}

	testrunDef.Status.Steps = tr.Testflow.Flow.GetStatuses()

	return wf, nil
}
