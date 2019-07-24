package copybird

import (
	"context"

	v1alpha1 "github.com/copybird/copybird-operator/pkg/apis/copybird/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	v1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_copybird")

// Add creates a new Copybird Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCopybird{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("copybird-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Copybird
	err = c.Watch(&source.Kind{Type: &v1alpha1.Copybird{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Copybird
	err = c.Watch(&source.Kind{Type: &v1beta1.CronJob{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.Copybird{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCopybird implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCopybird{}

// ReconcileCopybird reconciles a Copybird object
type ReconcileCopybird struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Copybird object and makes changes based on the state read
// and what is in the Copybird.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCopybird) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Copybird")

	// Fetch the Copybird instance
	instance := &v1alpha1.Copybird{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new CronJob object
	cronJob := newCronJobForCR(instance)

	// Set Copybird instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, cronJob, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this CronJob already exists
	found := &v1beta1.CronJob{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: cronJob.Name, Namespace: cronJob.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new CronJob", "CronJob.Namespace", cronJob.Namespace, "CronJob.Name", cronJob.Name)
		err = r.client.Create(context.TODO(), cronJob)
		if err != nil {
			return reconcile.Result{}, err
		}

		// CronJob created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	if found.Spec.Schedule != instance.Spec.Cron {
		found.Spec.Schedule = instance.Spec.Cron
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update CronJob.", "CronJob.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// CronJob already exists - don't requeue
	reqLogger.Info("Skip reconcile: CronJob already exists", "CronJob.Namespace", found.Namespace, "CronJob.Name", found.Name)
	return reconcile.Result{}, nil
}

// newCronJobForCR returns a busybox pod with the same name/namespace as the cr
func newCronJobForCR(cr *v1alpha1.Copybird) *v1beta1.CronJob {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &v1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-cronjob",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: v1beta1.CronJobSpec{
			Schedule: cr.Spec.Cron,
			JobTemplate: v1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      cr.Name + "-copybird",
							Namespace: cr.Namespace,
							Labels:    labels,
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:    cr.Name,
									Image:   "copybird/copybird",
									Command: []string{},
									Args:    []string{},
								},
							},
							RestartPolicy: "OnFailure",
						},
					},
				},
			},
		},
	}
}
