/*
Copyright 2026 Red Hat Inc.

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

// TODO: this entire package can be removed when we remove support for the old apiGroup
package conversion

import (
	newapi "github.com/konflux-ci/application-api/api/konflux/v1alpha1"
	oldapi "github.com/konflux-ci/application-api/api/v1alpha1"
)

func ConvertNewToOld(src newapi.Component) oldapi.Component {
	dst := oldapi.Component{
		ObjectMeta: *src.ObjectMeta.DeepCopy(),
		Spec: oldapi.ComponentSpec{
			Source: oldapi.ComponentSource{
				ComponentSourceUnion: oldapi.ComponentSourceUnion{
					GitSource: &oldapi.GitSource{
						URL: src.Spec.Source.GitURL,
					},
					GitURL:        src.Spec.Source.GitURL,
					DockerfileURI: src.Spec.Source.DockerfilePath,
					Versions:      convertVersionsNewToOld(src.Spec.Source.Versions),
				},
			},
			ContainerImage:       src.Spec.ContainerImage,
			Actions:              convertActionsNewToOld(src.Spec.Actions),
			SkipOffboardingPr:    src.Spec.SkipOffboardingPr,
			RepositorySettings:   convertRepoSettingsNewToOld(src.Spec.RepositorySettings),
			DefaultBuildPipeline: convertBuildPipelineNewToOld(src.Spec.DefaultBuildPipeline),
		},
		Status: oldapi.ComponentStatus{
			RepositorySettings: convertRepoSettingsNewToOld(src.Status.RepositorySettings),
			Message:            src.Status.Message,
			PacRepository:      src.Status.PacRepository,
			Versions:           convertVersionStatusesNewToOld(src.Status.Versions),
		},
	}
	return dst
}

func ConvertOldToNew(src oldapi.Component) newapi.Component {
	gitURL := src.Spec.Source.ComponentSourceUnion.GitURL
	if gitURL == "" && src.Spec.Source.ComponentSourceUnion.GitSource != nil {
		gitURL = src.Spec.Source.ComponentSourceUnion.GitSource.URL
	}

	dst := newapi.Component{
		ObjectMeta: *src.ObjectMeta.DeepCopy(),
		Spec: newapi.ComponentSpec{
			Source: newapi.ComponentSource{
				GitURL:         gitURL,
				DockerfilePath: src.Spec.Source.ComponentSourceUnion.DockerfileURI,
				Versions:       convertVersionsOldToNew(src.Spec.Source.ComponentSourceUnion.Versions),
			},
			ContainerImage:       src.Spec.ContainerImage,
			Actions:              convertActionsOldToNew(src.Spec.Actions),
			SkipOffboardingPr:    src.Spec.SkipOffboardingPr,
			RepositorySettings:   convertRepoSettingsOldToNew(src.Spec.RepositorySettings),
			DefaultBuildPipeline: convertBuildPipelineOldToNew(src.Spec.DefaultBuildPipeline),
		},
		Status: newapi.ComponentStatus{
			RepositorySettings: convertRepoSettingsOldToNew(src.Status.RepositorySettings),
			Message:            src.Status.Message,
			PacRepository:      src.Status.PacRepository,
			Versions:           convertVersionStatusesOldToNew(src.Status.Versions),
		},
	}
	return dst
}

// --- Spec helpers ---

func convertVersionsNewToOld(versions []newapi.ComponentVersion) []oldapi.ComponentVersion {
	if versions == nil {
		return nil
	}
	out := make([]oldapi.ComponentVersion, len(versions))
	for i, v := range versions {
		out[i] = oldapi.ComponentVersion{
			BuildPipeline: convertBuildPipelineNewToOld(v.BuildPipeline),
			Context:       v.Context,
			DockerfileURI: v.DockerfilePath,
			Name:          v.Name,
			Revision:      v.Revision,
			SkipBuilds:    v.SkipBuilds,
		}
	}
	return out
}

func convertVersionsOldToNew(versions []oldapi.ComponentVersion) []newapi.ComponentVersion {
	if versions == nil {
		return nil
	}
	out := make([]newapi.ComponentVersion, len(versions))
	for i, v := range versions {
		out[i] = newapi.ComponentVersion{
			BuildPipeline:  convertBuildPipelineOldToNew(v.BuildPipeline),
			Context:        v.Context,
			DockerfilePath: v.DockerfileURI,
			Name:           v.Name,
			Revision:       v.Revision,
			SkipBuilds:     v.SkipBuilds,
		}
	}
	return out
}

func convertActionsNewToOld(src newapi.ComponentActions) oldapi.ComponentActions {
	return oldapi.ComponentActions{
		CreateConfiguration: oldapi.ComponentCreatePipelineConfiguration{
			AllVersions: src.CreateConfiguration.AllVersions,
			Version:     src.CreateConfiguration.Version,
			Versions:    src.CreateConfiguration.Versions,
		},
		TriggerBuild:  src.TriggerBuild,
		TriggerBuilds: src.TriggerBuilds,
	}
}

func convertActionsOldToNew(src oldapi.ComponentActions) newapi.ComponentActions {
	return newapi.ComponentActions{
		CreateConfiguration: newapi.ComponentCreatePipelineConfiguration{
			AllVersions: src.CreateConfiguration.AllVersions,
			Version:     src.CreateConfiguration.Version,
			Versions:    src.CreateConfiguration.Versions,
		},
		TriggerBuild:  src.TriggerBuild,
		TriggerBuilds: src.TriggerBuilds,
	}
}

func convertRepoSettingsNewToOld(src newapi.RepositorySettings) oldapi.RepositorySettings {
	return oldapi.RepositorySettings{
		CommentStrategy:          src.CommentStrategy,
		GithubAppTokenScopeRepos: src.GithubAppTokenScopeRepos,
	}
}

func convertRepoSettingsOldToNew(src oldapi.RepositorySettings) newapi.RepositorySettings {
	return newapi.RepositorySettings{
		CommentStrategy:          src.CommentStrategy,
		GithubAppTokenScopeRepos: src.GithubAppTokenScopeRepos,
	}
}

func convertBuildPipelineNewToOld(src *newapi.ComponentBuildPipeline) *oldapi.ComponentBuildPipeline {
	if src == nil {
		return nil
	}
	return &oldapi.ComponentBuildPipeline{
		PullAndPush: convertPipelineDefNewToOld(src.PullAndPush),
		Pull:        convertPipelineDefNewToOld(src.Pull),
		Push:        convertPipelineDefNewToOld(src.Push),
	}
}

func convertBuildPipelineOldToNew(src *oldapi.ComponentBuildPipeline) *newapi.ComponentBuildPipeline {
	if src == nil {
		return nil
	}
	return &newapi.ComponentBuildPipeline{
		PullAndPush: convertPipelineDefOldToNew(src.PullAndPush),
		Pull:        convertPipelineDefOldToNew(src.Pull),
		Push:        convertPipelineDefOldToNew(src.Push),
	}
}

func convertPipelineDefNewToOld(src *newapi.PipelineDefinition) *oldapi.PipelineDefinition {
	if src == nil {
		return nil
	}
	return &oldapi.PipelineDefinition{
		PipelineRefGit:         convertPipelineRefGitNewToOld(src.PipelineRefGit),
		PipelineRefName:        src.PipelineRefName,
		PipelineSpecFromBundle: convertPipelineSpecBundleNewToOld(src.PipelineSpecFromBundle),
	}
}

func convertPipelineDefOldToNew(src *oldapi.PipelineDefinition) *newapi.PipelineDefinition {
	if src == nil {
		return nil
	}
	return &newapi.PipelineDefinition{
		PipelineRefGit:         convertPipelineRefGitOldToNew(src.PipelineRefGit),
		PipelineRefName:        src.PipelineRefName,
		PipelineSpecFromBundle: convertPipelineSpecBundleOldToNew(src.PipelineSpecFromBundle),
	}
}

func convertPipelineRefGitNewToOld(src *newapi.PipelineRefGit) *oldapi.PipelineRefGit {
	if src == nil {
		return nil
	}
	return &oldapi.PipelineRefGit{
		PathInRepo: src.PathInRepo,
		Revision:   src.Revision,
		Url:        src.Url,
	}
}

func convertPipelineRefGitOldToNew(src *oldapi.PipelineRefGit) *newapi.PipelineRefGit {
	if src == nil {
		return nil
	}
	return &newapi.PipelineRefGit{
		PathInRepo: src.PathInRepo,
		Revision:   src.Revision,
		Url:        src.Url,
	}
}

func convertPipelineSpecBundleNewToOld(src *newapi.PipelineSpecFromBundle) *oldapi.PipelineSpecFromBundle {
	if src == nil {
		return nil
	}
	return &oldapi.PipelineSpecFromBundle{
		Bundle: src.Bundle,
		Name:   src.Name,
	}
}

func convertPipelineSpecBundleOldToNew(src *oldapi.PipelineSpecFromBundle) *newapi.PipelineSpecFromBundle {
	if src == nil {
		return nil
	}
	return &newapi.PipelineSpecFromBundle{
		Bundle: src.Bundle,
		Name:   src.Name,
	}
}

// --- Status helpers ---

func convertVersionStatusesNewToOld(versions []newapi.ComponentVersionStatus) []oldapi.ComponentVersionStatus {
	if versions == nil {
		return nil
	}
	out := make([]oldapi.ComponentVersionStatus, len(versions))
	for i, v := range versions {
		out[i] = oldapi.ComponentVersionStatus{
			ConfigurationMergeURL: v.ConfigurationMergeURL,
			Message:               v.Message,
			Name:                  v.Name,
			OnboardingStatus:      v.OnboardingStatus,
			OnboardingTime:        v.OnboardingTime,
			Revision:              v.Revision,
			SkipBuilds:            v.SkipBuilds,
		}
	}
	return out
}

func convertVersionStatusesOldToNew(versions []oldapi.ComponentVersionStatus) []newapi.ComponentVersionStatus {
	if versions == nil {
		return nil
	}
	out := make([]newapi.ComponentVersionStatus, len(versions))
	for i, v := range versions {
		out[i] = newapi.ComponentVersionStatus{
			ConfigurationMergeURL: v.ConfigurationMergeURL,
			Message:               v.Message,
			Name:                  v.Name,
			OnboardingStatus:      v.OnboardingStatus,
			OnboardingTime:        v.OnboardingTime,
			Revision:              v.Revision,
			SkipBuilds:            v.SkipBuilds,
		}
	}
	return out
}
