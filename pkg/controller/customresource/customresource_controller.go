package customresource

import (
	"context"

	"github.com/go-logr/logr"
	cachev1alpha1 "github.com/ironoa/kubernetes-customresource-operator/pkg/apis/cache/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_customresource")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CustomResource Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCustomResource{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("customresource-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CustomResource
	err = c.Watch(&source.Kind{Type: &cachev1alpha1.CustomResource{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Deployments and requeue the owner CustomResource
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.CustomResource{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCustomResource implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCustomResource{}

// ReconcileCustomResource reconciles a CustomResource object
type ReconcileCustomResource struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CustomResource object and makes changes based on the state read
// and what is in the CustomResource.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCustomResource) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	logger.Info("Reconciling CustomResource")

	instance, err := r.fetchCustomResource(request)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("Custom Resource not found... Return and not requeing the request")
			return reconcile.Result{}, nil
		}
		logger.Error(err, "Error on fetch the Custom Resource... Requeing the Reconciling request... ")
		return reconcile.Result{}, err
	}

	err = r.handleDeployment(instance)
	if err != nil {
		logger.Info("Requeing the Reconciling request... ")
		if errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, nil
		}
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileCustomResource) fetchCustomResource(request reconcile.Request) (*cachev1alpha1.CustomResource, error) {
	instance := &cachev1alpha1.CustomResource{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	return instance, err
}

func (r *ReconcileCustomResource) handleDeployment(instance *cachev1alpha1.CustomResource) error {
	desideredDeployment := r.newDeploymentForCR(instance)
	logger := log.WithValues("Deployment.Namespace", desideredDeployment.Namespace, "Deployment.Name", desideredDeployment.Name)

	foundDeployment, err := r.fetchDeployment(desideredDeployment)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Error on fetch the Deployment...")
			return err
		}
		logger.Info("Deployment not found...")
		creationError := r.createDeployment(desideredDeployment, logger)
		if creationError != nil {
			logger.Error(creationError, "Error on creating a new Deployment...")
			return creationError
		}
		logger.Info("Created the new Deployment")
		return err
	}

	if areDeploymentsDifferent(foundDeployment, desideredDeployment, logger) {
		err := r.updateDeployment(desideredDeployment, logger)
		if err != nil {
			logger.Error(err, "Update Deployment Error...")
			return err
		}
		logger.Info("Updated the Deployment...")
	}

	return nil
}

func (r *ReconcileCustomResource) fetchDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	found := &appsv1.Deployment{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	return found, err
}

func (r *ReconcileCustomResource) createDeployment(deployment *appsv1.Deployment, logger logr.Logger) error {
	logger.Info("Creating a new Deployment...")
	err := r.client.Create(context.TODO(), deployment)
	return err
}

func areDeploymentsDifferent(currentDeployment *appsv1.Deployment, desideredDeployment *appsv1.Deployment, logger logr.Logger) bool {
	result := false

	if isDeploymentReplicaDifferent(currentDeployment, desideredDeployment, logger) {
		result = true
	}
	if isDeploymentVersionDifferent(currentDeployment, desideredDeployment, logger) {
		result = true
	}

	return result
}

/* TODO investigate, now it is not working
func (r *ReconcileCustomResource) alignDeployments(foundDeployment *appsv1.Deployment, desideredDeployment *appsv1.Deployment, logger logr.Logger) error {

	//deep check
	logger.Info("Comparing found and desidered....")
	//if *foundDeployment != *desideredDeployment {
	//	logger.Info("Found a difference: normal check")
	//}

	diff := deep.Equal(*foundDeployment, *desideredDeployment)
	if diff != nil {
		logger.Info("Found a difference: library check")
		logger.Info("differences:", "Differences:", diff)
	}

	//update

	return nil
}*/

func isDeploymentReplicaDifferent(currentDeployment *appsv1.Deployment, desideredDeployment *appsv1.Deployment, logger logr.Logger) bool {
	size := *desideredDeployment.Spec.Replicas
	if *currentDeployment.Spec.Replicas != size {
		logger.Info("Find a replica size mismatch...")
		return true
	}
	return false
}

func isDeploymentVersionDifferent(currentDeployment *appsv1.Deployment, desideredDeployment *appsv1.Deployment, logger logr.Logger) bool {
	version := desideredDeployment.ObjectMeta.Labels["version"]
	if currentDeployment.ObjectMeta.Labels["version"] != version {
		logger.Info("Found a version mismatch...")
		return true
	}
	return false
}

func (r *ReconcileCustomResource) updateDeployment(deployment *appsv1.Deployment, logger logr.Logger) error {
	logger.Info("Updating the Deployment...")
	return r.client.Update(context.TODO(), deployment)
}

func (r *ReconcileCustomResource) newDeploymentForCR(cr *cachev1alpha1.CustomResource) *appsv1.Deployment {
	labels := labelsForApp(cr)
	replicas := cr.Spec.Size
	version := cr.Spec.Version
	labelsWithVersion := labelsForAppWithVersion(cr, version)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-deployment",
			Namespace: cr.Namespace,
			Labels:    labelsWithVersion,
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
					Containers: []corev1.Container{{
						Name:    "busybox",
						Image:   "busybox:" + version,
						Command: []string{"sleep", "3600"},
					}},
				},
			},
		},
	}

	// Set App instance as the owner and controller.
	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	controllerutil.SetControllerReference(cr, deployment, r.scheme)
	return deployment
}

func (r *ReconcileCustomResource) newDeploymentForCRLivenessTest(cr *cachev1alpha1.CustomResource) *appsv1.Deployment {
	labels := labelsForApp(cr)
	replicas := cr.Spec.Size
	version := cr.Spec.Version
	labelsWithVersion := labelsForAppWithVersion(cr, version)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-deployment",
			Namespace: cr.Namespace,
			Labels:    labelsWithVersion,
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
					Containers: []corev1.Container{{
						Name:  "busybox-liveness",
						Image: "k8s.gcr.io/busybox:" + cr.Spec.Version,
						Args:  []string{"/bin/sh", "-c", "touch /tmp/healthy; sleep 30; rm -rf /tmp/healthy; sleep 600"},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"cat", "/tmp/healthy"},
								},
							},
							InitialDelaySeconds: 3,
							PeriodSeconds:       5,
						},
					}},
				},
			},
		},
	}

	// Set App instance as the owner and controller.
	// NOTE: calling SetControllerReference, and setting owner references in
	// general, is important as it allows deleted objects to be garbage collected.
	controllerutil.SetControllerReference(cr, deployment, r.scheme)
	return deployment
}

// labelsForApp creates a simple set of labels for App.
func labelsForApp(cr *cachev1alpha1.CustomResource) map[string]string {
	return map[string]string{"app_name": "app", "app_cr": cr.Name}
}

func labelsForAppWithVersion(cr *cachev1alpha1.CustomResource, version string) map[string]string {
	labels := labelsForApp(cr)
	labels["version"] = version
	return labels
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *cachev1alpha1.CustomResource) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
