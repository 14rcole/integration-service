package snapshot

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	"github.com/konflux-ci/integration-service/api/v1beta2"
	"github.com/konflux-ci/integration-service/gitops"
	"github.com/konflux-ci/integration-service/helpers"
	"github.com/konflux-ci/integration-service/loader"
	"github.com/konflux-ci/integration-service/tekton"
	tektonv1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// prepareSnapshotForPipelineRun prepares the Snapshot for a given PipelineRun,
// component and application. In case the Snapshot can't be created, an error will be returned.
func PrepareSnapshotForPipelineRun(pipelineRun *tektonv1.PipelineRun, component *applicationapiv1alpha1.Component, componentGroup *v1beta2.ComponentGroup, loader loader.ObjectLoader) (*applicationapiv1alpha1.Snapshot, error) {
	newContainerImage, err := tekton.GetImagePullSpecFromPipelineRun(pipelineRun)
	if err != nil {
		return nil, err
	}
	componentSource, err := tekton.GetComponentSourceFromPipelineRun(pipelineRun)
	if err != nil {
		return nil, err
	}

	// TODO: create new loader function
	groupComponents, err := loader.GetAllComponentsInComponentGroup(a.context, a.client, componentGroup)
	if err != nil {
		return nil, err
	}

	// TODO: create PrepareSnapshotApplication
	snapshot, err := PrepareSnapshot(a.context, a.client, componentGroup, groupComponents, component, newContainerImage, componentSource)
	if err != nil {
		return nil, err
	}

	prefixes := []string{gitops.BuildPipelineRunPrefix, gitops.TestLabelPrefix, gitops.CustomLabelPrefix, gitops.ReleaseLabelPrefix}
	CopySnapshotLabelsAndAnnotations(componentGroup, snapshot, component.Name, &pipelineRun.ObjectMeta, prefixes, false)

	snapshot.Labels[gitops.BuildPipelineRunNameLabel] = pipelineRun.Name
	if pipelineRun.Status.CompletionTime != nil {
		snapshot.Labels[gitops.BuildPipelineRunFinishTimeLabel] = strconv.FormatInt(pipelineRun.Status.CompletionTime.Unix(), 10)
	} else {
		snapshot.Labels[gitops.BuildPipelineRunFinishTimeLabel] = strconv.FormatInt(time.Now().Unix(), 10)
	}

	if IsSnapshotCreatedByPACMergeQueueEvent(snapshot) {
		pullRequestNumber := ExtractPullRequestNumberFromMergeQueueSnapshot(snapshot)
		if pullRequestNumber != "" {
			snapshot.Labels[gitops.PipelineAsCodePullRequestAnnotation] = pullRequestNumber
			snapshot.Annotations[gitops.PipelineAsCodePullRequestAnnotation] = pullRequestNumber
		}
	}

	// Set BuildPipelineRunStartTime annotation with millisecond precision and override snapshot name
	var timestampMillis int64
	// Get the time
	if pipelineRun.Status.StartTime != nil {
		timestampMillis = pipelineRun.Status.StartTime.UnixMilli()
	} else {
		timestampMillis = time.Now().UnixMilli()
	}
	// Naming once, at the end
	snapshot.Annotations[gitops.BuildPipelineRunStartTime] = strconv.FormatInt(timestampMillis, 10)
	snapshot.Name = GenerateSnapshotNameWithTimestamp(application.Name, timestampMillis)

	// Set the integration workflow annotation based on the PipelineRun type
	if tekton.IsPLRCreatedByPACPushEvent(pipelineRun) {
		snapshot.Annotations[gitops.IntegrationWorkflowAnnotation] = IntegrationWorkflowPushValue
	} else {
		snapshot.Annotations[gitops.IntegrationWorkflowAnnotation] = IntegrationWorkflowPullRequestValue
	}

	return snapshot, nil
}

// CreateSnapshotWithCollisionHandling attempts to create a snapshot, retrying with a random suffix if collision occurs
func CreateSnapshotWithCollisionHandling(ctx context.Context, client client.Client, pipelineRun *tektonv1.PipelineRun, snapshot *applicationapiv1alpha1.Snapshot, componentGroup *v1beta2.ComponentGroup, logger helpers.IntegrationLogger) error {
	originalName := snapshot.Name
	maxRetries := 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := client.Create(ctx, snapshot)
		if err == nil {
			// Success
			if attempt > 0 {
				logger.Info("Successfully created snapshot after collision retry",
					"originalName", originalName,
					"finalName", snapshot.Name,
					"attempts", attempt+1)
			}
			return nil
		}

		// Check if it's an "already exists" error
		if !errors.IsAlreadyExists(err) {
			// Not a collision error, return immediately
			return err
		}

		// Collision detected - generate new name with suffix
		if attempt < maxRetries-1 {
			suffix, suffixErr := generateRandomSuffix()
			if suffixErr != nil {
				logger.Error(suffixErr, "Failed to generate random suffix for snapshot name collision")
				return err // Return original collision error
			}

			// Extract timestamp from original name or use current time
			var timestampMillis int64
			if pipelineRun.Status.StartTime != nil {
				timestampMillis = pipelineRun.Status.StartTime.UnixMilli()
			} else {
				timestampMillis = time.Now().UnixMilli()
			}

			// Regenerate name with suffix
			snapshot.Name = GenerateSnapshotNameWithTimestamp(componentGroup.Name, timestampMillis, suffix)
			logger.Info("Snapshot name collision detected, retrying with suffix",
				"originalName", originalName,
				"newName", snapshot.Name,
				"attempt", attempt+1,
				"maxRetries", maxRetries)
		} else {
			// Max retries reached
			logger.Error(err, "Failed to create snapshot after max retries due to collisions",
				"originalName", originalName,
				"attempts", maxRetries)
			return err
		}
	}

	return fmt.Errorf("failed to create snapshot after %d attempts", maxRetries)
}

// generateRandomSuffix generates a random 2-character alphanumeric suffix for collision handling
func generateRandomSuffix() (string, error) {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyz"
	const suffixLength = 2
	suffix := make([]byte, suffixLength)
	for i := range suffix {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		suffix[i] = charset[num.Int64()]
	}
	return string(suffix), nil
}
