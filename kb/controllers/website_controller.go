/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	websitev1 "CustomerController/websites/kb/api/v1"
)

// WebsiteReconciler reconciles a Website object
type WebsiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kb.crd.dango.io,resources=websites,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kb.crd.dango.io,resources=websites/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kb.crd.dango.io,resources=websites/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Website object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *WebsiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	res := ctrl.Result{}

	// TODO(user): your logic here
	var website websitev1.Website
	err := r.Get(ctx, req.NamespacedName, &website)
	if err != nil {
		// The Foo resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			return res, fmt.Errorf("website '%s' in work queue no longer exists", req.Name)
		}

		res.Requeue = true
		return res, fmt.Errorf("website '%s' in work queue no longer exists", req.Name)
	}

	// 检查 deployment 是否存在，不存在就创建
	deploymentName := "deployment-" + website.Name
	// Get the deployment with the name specified in Foo.spec

	var deployment appsv1.Deployment
	err = r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      deploymentName,
	}, &deployment)
	// If the resource doesn't exist, we'll create it
	if errors.IsNotFound(err) {
		deployment = newDeployment(&website)
		logger.Info(fmt.Sprintf("deploy: %v", deployment))
		err = r.Create(ctx, &deployment)
		logger.Info("Create a deployment")
	}

	// If an error occurs during Get/Create, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		logger.Error(err, "Create a deployment fail")
		return res, err
	}

	// If this number of the replicas on the Foo resource is specified, and the
	// number does not equal the current desired replicas on the Deployment, we
	// should update the Deployment resource.
	if *deployment.Spec.Replicas != 1 {
		logger.V(4).Info(fmt.Sprintf("Website %s replicas: %d, deployment replicas: %d", req.Name, 1, *deployment.Spec.Replicas))
		d := newDeployment(&website)
		r.Update(ctx, &d)
	}

	// If an error occurs during Update, we'll requeue the item so we can
	// attempt processing again later. This could have been caused by a
	// temporary network failure, or any other transient reason.
	if err != nil {
		return res, err
	}

	// 查看 service 是否存在，不存在就创建
	serviceName := "service-" + website.Name
	var service corev1.Service
	err = r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      serviceName,
	}, &service)

	if errors.IsNotFound(err) {
		service = newService(&website)
		err = r.Create(ctx, &service)
		logger.Info("Create a Service")
	}

	if err != nil {
		logger.Error(err, "Create a service fail")
		return res, nil
	}
	logger.Info(fmt.Sprintf("service's port: %d", service.Spec.Ports[0].NodePort))

	// Finally, we update the status block of the Foo resource to reflect the
	// current state of the world
	err = r.updateWebsiteStatus(ctx, &website, &deployment)
	if err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WebsiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&websitev1.Website{}).
		Complete(r)
}
func (c *WebsiteReconciler) updateWebsiteStatus(ctx context.Context, website *websitev1.Website, deployment *appsv1.Deployment) error {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	websiteCp := website.DeepCopy()
	websiteCp.Status.AvailableReplicas = deployment.Status.AvailableReplicas
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the Foo resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	err := c.Update(ctx, websiteCp)
	return err
}

// newDeployment creates a new Deployment for a Foo resource. It also sets
// the appropriate OwnerReferences on the resource so handleObject can discover
// the Foo resource that 'owns' it.
func newDeployment(website *websitev1.Website) appsv1.Deployment {
	labels := map[string]string{
		"app":        "web",
		"controller": website.Name,
	}
	var replicas int32 = 1
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deployment-" + website.Name,
			Namespace: website.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(website, websitev1.GroupVersion.WithKind("Website")),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "gitsync",
							Image: "bitnami/git",
							Command: []string{
								"sh",
								"-c",
								"cd /tmp/git;git clone " + website.Spec.GitRepo,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "gitrepo",
									MountPath: "/tmp/git",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "gitrepo",
									MountPath: "/usr/share/nginx/html",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "gitrepo",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

func newService(website *websitev1.Website) corev1.Service {
	labels := map[string]string{
		"app":        "web",
		"controller": website.Name,
	}
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "service-" + website.Name,
			Namespace: website.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(website, websitev1.GroupVersion.WithKind("Website")),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeNodePort,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
					NodePort:   int32(website.Spec.Port),
				},
			},
		},
	}
}
