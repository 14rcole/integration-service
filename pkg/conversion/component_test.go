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

package conversion

import (
	"testing"

	newapi "github.com/konflux-ci/application-api/api/konflux/v1alpha1"
	oldapi "github.com/konflux-ci/application-api/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertNewToOld_FullComponent(t *testing.T) {
	src := newapi.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-component",
			Namespace:       "default",
			ResourceVersion: "123",
			Annotations: map[string]string{
				"test-key": "test-value",
			},
		},
		Spec: newapi.ComponentSpec{
			Source: newapi.ComponentSource{
				GitURL:         "https://github.com/org/repo",
				DockerfilePath: "docker/Dockerfile",
				Versions: []newapi.ComponentVersion{
					{
						Name:           "v1",
						Revision:       "main",
						Context:        "src",
						DockerfilePath: "src/Dockerfile",
						SkipBuilds:     false,
						BuildPipeline: &newapi.ComponentBuildPipeline{
							PullAndPush: &newapi.PipelineDefinition{
								PipelineRefName: "my-pipeline",
							},
						},
					},
					{
						Name:       "v2",
						Revision:   "release-2.0",
						SkipBuilds: true,
					},
				},
			},
			ContainerImage:    "quay.io/org/repo",
			SkipOffboardingPr: true,
			RepositorySettings: newapi.RepositorySettings{
				CommentStrategy:          "always",
				GithubAppTokenScopeRepos: []string{"extra-repo"},
			},
			DefaultBuildPipeline: &newapi.ComponentBuildPipeline{
				Push: &newapi.PipelineDefinition{
					PipelineSpecFromBundle: &newapi.PipelineSpecFromBundle{
						Bundle: "latest",
						Name:   "docker-build",
					},
				},
				Pull: &newapi.PipelineDefinition{
					PipelineRefGit: &newapi.PipelineRefGit{
						PathInRepo: "pipeline/pull.yaml",
						Revision:   "main",
						Url:        "https://github.com/pipelines/repo",
					},
				},
			},
			Actions: newapi.ComponentActions{
				TriggerBuild:  "v1",
				TriggerBuilds: []string{"v2"},
				CreateConfiguration: newapi.ComponentCreatePipelineConfiguration{
					AllVersions: true,
				},
			},
		},
		Status: newapi.ComponentStatus{
			Message:       "all good",
			PacRepository: "my-component-pac",
			RepositorySettings: newapi.RepositorySettings{
				CommentStrategy: "always",
			},
			Versions: []newapi.ComponentVersionStatus{
				{
					Name:                  "v1",
					Revision:              "main",
					OnboardingStatus:      "succeeded",
					OnboardingTime:        "01 Jan 2026 00:00:00 UTC",
					ConfigurationMergeURL: "https://github.com/org/repo/pull/1",
				},
			},
		},
	}

	dst := ConvertNewToOld(src)

	// ObjectMeta
	if dst.Name != "my-component" {
		t.Errorf("Name: got %q, want %q", dst.Name, "my-component")
	}
	if dst.Namespace != "default" {
		t.Errorf("Namespace: got %q, want %q", dst.Namespace, "default")
	}
	if dst.ResourceVersion != "123" {
		t.Errorf("ResourceVersion: got %q, want %q", dst.ResourceVersion, "123")
	}
	if dst.Annotations["test-key"] != "test-value" {
		t.Errorf("Annotations[test-key]: got %q, want %q", dst.Annotations["test-key"], "test-value")
	}

	// Source
	if dst.Spec.Source.ComponentSourceUnion.GitURL != "https://github.com/org/repo" {
		t.Errorf("GitURL: got %q", dst.Spec.Source.ComponentSourceUnion.GitURL)
	}
	if dst.Spec.Source.ComponentSourceUnion.GitSource == nil {
		t.Fatal("GitSource should not be nil")
	}
	if dst.Spec.Source.ComponentSourceUnion.GitSource.URL != "https://github.com/org/repo" {
		t.Errorf("GitSource.URL: got %q", dst.Spec.Source.ComponentSourceUnion.GitSource.URL)
	}
	if dst.Spec.Source.ComponentSourceUnion.DockerfileURI != "docker/Dockerfile" {
		t.Errorf("DockerfileURI: got %q", dst.Spec.Source.ComponentSourceUnion.DockerfileURI)
	}

	// Versions
	if len(dst.Spec.Source.ComponentSourceUnion.Versions) != 2 {
		t.Fatalf("Versions count: got %d, want 2", len(dst.Spec.Source.ComponentSourceUnion.Versions))
	}
	v1 := dst.Spec.Source.ComponentSourceUnion.Versions[0]
	if v1.Name != "v1" || v1.Revision != "main" || v1.Context != "src" {
		t.Errorf("Version[0]: got name=%q revision=%q context=%q", v1.Name, v1.Revision, v1.Context)
	}
	if v1.DockerfileURI != "src/Dockerfile" {
		t.Errorf("Version[0] DockerfileURI: got %q, want %q", v1.DockerfileURI, "src/Dockerfile")
	}
	if v1.BuildPipeline == nil || v1.BuildPipeline.PullAndPush == nil {
		t.Fatal("Version[0] BuildPipeline.PullAndPush should not be nil")
	}
	if v1.BuildPipeline.PullAndPush.PipelineRefName != "my-pipeline" {
		t.Errorf("Version[0] PipelineRefName: got %q", v1.BuildPipeline.PullAndPush.PipelineRefName)
	}
	v2 := dst.Spec.Source.ComponentSourceUnion.Versions[1]
	if v2.Name != "v2" || !v2.SkipBuilds {
		t.Errorf("Version[1]: got name=%q skipBuilds=%v", v2.Name, v2.SkipBuilds)
	}

	// ContainerImage
	if dst.Spec.ContainerImage != "quay.io/org/repo" {
		t.Errorf("ContainerImage: got %q", dst.Spec.ContainerImage)
	}

	// Actions
	if dst.Spec.Actions.TriggerBuild != "v1" {
		t.Errorf("Actions.TriggerBuild: got %q", dst.Spec.Actions.TriggerBuild)
	}
	if len(dst.Spec.Actions.TriggerBuilds) != 1 || dst.Spec.Actions.TriggerBuilds[0] != "v2" {
		t.Errorf("Actions.TriggerBuilds: got %v", dst.Spec.Actions.TriggerBuilds)
	}
	if !dst.Spec.Actions.CreateConfiguration.AllVersions {
		t.Error("Actions.CreateConfiguration.AllVersions should be true")
	}

	// SkipOffboardingPr
	if !dst.Spec.SkipOffboardingPr {
		t.Error("SkipOffboardingPr should be true")
	}

	// RepositorySettings
	if dst.Spec.RepositorySettings.CommentStrategy != "always" {
		t.Errorf("RepositorySettings.CommentStrategy: got %q", dst.Spec.RepositorySettings.CommentStrategy)
	}
	if len(dst.Spec.RepositorySettings.GithubAppTokenScopeRepos) != 1 {
		t.Errorf("RepositorySettings.GithubAppTokenScopeRepos: got %v", dst.Spec.RepositorySettings.GithubAppTokenScopeRepos)
	}

	// DefaultBuildPipeline
	if dst.Spec.DefaultBuildPipeline == nil {
		t.Fatal("DefaultBuildPipeline should not be nil")
	}
	if dst.Spec.DefaultBuildPipeline.Push == nil || dst.Spec.DefaultBuildPipeline.Push.PipelineSpecFromBundle == nil {
		t.Fatal("DefaultBuildPipeline.Push.PipelineSpecFromBundle should not be nil")
	}
	if dst.Spec.DefaultBuildPipeline.Push.PipelineSpecFromBundle.Bundle != "latest" {
		t.Errorf("Push bundle: got %q", dst.Spec.DefaultBuildPipeline.Push.PipelineSpecFromBundle.Bundle)
	}
	if dst.Spec.DefaultBuildPipeline.Pull == nil || dst.Spec.DefaultBuildPipeline.Pull.PipelineRefGit == nil {
		t.Fatal("DefaultBuildPipeline.Pull.PipelineRefGit should not be nil")
	}
	if dst.Spec.DefaultBuildPipeline.Pull.PipelineRefGit.Url != "https://github.com/pipelines/repo" {
		t.Errorf("Pull PipelineRefGit.Url: got %q", dst.Spec.DefaultBuildPipeline.Pull.PipelineRefGit.Url)
	}

	// Status
	if dst.Status.Message != "all good" {
		t.Errorf("Status.Message: got %q", dst.Status.Message)
	}
	if dst.Status.PacRepository != "my-component-pac" {
		t.Errorf("Status.PacRepository: got %q", dst.Status.PacRepository)
	}
	if dst.Status.RepositorySettings.CommentStrategy != "always" {
		t.Errorf("Status.RepositorySettings.CommentStrategy: got %q", dst.Status.RepositorySettings.CommentStrategy)
	}
	if len(dst.Status.Versions) != 1 {
		t.Fatalf("Status.Versions count: got %d", len(dst.Status.Versions))
	}
	sv := dst.Status.Versions[0]
	if sv.Name != "v1" || sv.OnboardingStatus != "succeeded" || sv.ConfigurationMergeURL != "https://github.com/org/repo/pull/1" {
		t.Errorf("Status.Versions[0]: got %+v", sv)
	}

	// Legacy fields should be zero-valued
	if dst.Spec.Application != "" {
		t.Errorf("Application should be empty, got %q", dst.Spec.Application)
	}
	if dst.Status.LastPromotedImage != "" {
		t.Errorf("LastPromotedImage should be empty, got %q", dst.Status.LastPromotedImage)
	}
	if dst.Status.LastBuiltCommit != "" {
		t.Errorf("LastBuiltCommit should be empty, got %q", dst.Status.LastBuiltCommit)
	}
}

func TestConvertNewToOld_EmptySource(t *testing.T) {
	src := newapi.Component{
		ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "ns"},
		Spec: newapi.ComponentSpec{
			Source: newapi.ComponentSource{},
		},
	}

	dst := ConvertNewToOld(src)

	if dst.Spec.Source.ComponentSourceUnion.GitSource == nil {
		t.Fatal("GitSource should not be nil even for empty URL")
	}
	if dst.Spec.Source.ComponentSourceUnion.GitSource.URL != "" {
		t.Errorf("GitSource.URL should be empty, got %q", dst.Spec.Source.ComponentSourceUnion.GitSource.URL)
	}
	if dst.Spec.Source.ComponentSourceUnion.Versions != nil {
		t.Errorf("Versions should be nil, got %v", dst.Spec.Source.ComponentSourceUnion.Versions)
	}
}

func TestConvertOldToNew_FullComponent(t *testing.T) {
	src := oldapi.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "old-component",
			Namespace:       "default",
			ResourceVersion: "456",
		},
		Spec: oldapi.ComponentSpec{
			Application: "my-app",
			Source: oldapi.ComponentSource{
				ComponentSourceUnion: oldapi.ComponentSourceUnion{
					GitSource: &oldapi.GitSource{
						URL:      "https://github.com/org/repo",
						Revision: "main",
						Context:  "src",
					},
					GitURL:        "https://github.com/org/repo",
					DockerfileURI: "Dockerfile",
					Versions: []oldapi.ComponentVersion{
						{
							Name:          "v1",
							Revision:      "main",
							Context:       "src",
							DockerfileURI: "src/Dockerfile",
						},
					},
				},
			},
			ContainerImage: "quay.io/org/repo",
			RepositorySettings: oldapi.RepositorySettings{
				CommentStrategy: "always",
			},
		},
		Status: oldapi.ComponentStatus{
			LastPromotedImage: "quay.io/org/repo@sha256:abc",
			LastBuiltCommit:   "abc123",
			Message:           "ok",
			PacRepository:     "pac-repo",
			Versions: []oldapi.ComponentVersionStatus{
				{Name: "v1", OnboardingStatus: "succeeded"},
			},
		},
	}

	dst := ConvertOldToNew(src)

	// ObjectMeta
	if dst.Name != "old-component" {
		t.Errorf("Name: got %q", dst.Name)
	}
	if dst.ResourceVersion != "456" {
		t.Errorf("ResourceVersion: got %q", dst.ResourceVersion)
	}

	// Source — prefers GitURL over GitSource.URL
	if dst.Spec.Source.GitURL != "https://github.com/org/repo" {
		t.Errorf("GitURL: got %q", dst.Spec.Source.GitURL)
	}
	if dst.Spec.Source.DockerfilePath != "Dockerfile" {
		t.Errorf("DockerfilePath: got %q", dst.Spec.Source.DockerfilePath)
	}

	// Versions
	if len(dst.Spec.Source.Versions) != 1 {
		t.Fatalf("Versions count: got %d", len(dst.Spec.Source.Versions))
	}
	if dst.Spec.Source.Versions[0].DockerfilePath != "src/Dockerfile" {
		t.Errorf("Versions[0].DockerfilePath: got %q", dst.Spec.Source.Versions[0].DockerfilePath)
	}

	// Status — legacy fields are dropped
	if dst.Status.Message != "ok" {
		t.Errorf("Status.Message: got %q", dst.Status.Message)
	}
	if len(dst.Status.Versions) != 1 {
		t.Fatalf("Status.Versions: got %d", len(dst.Status.Versions))
	}
}

func TestConvertOldToNew_GitSourceFallback(t *testing.T) {
	src := oldapi.Component{
		ObjectMeta: metav1.ObjectMeta{Name: "fallback", Namespace: "ns"},
		Spec: oldapi.ComponentSpec{
			Source: oldapi.ComponentSource{
				ComponentSourceUnion: oldapi.ComponentSourceUnion{
					GitSource: &oldapi.GitSource{
						URL: "https://github.com/org/from-gitsource",
					},
				},
			},
		},
	}

	dst := ConvertOldToNew(src)

	if dst.Spec.Source.GitURL != "https://github.com/org/from-gitsource" {
		t.Errorf("Expected GitURL from GitSource fallback, got %q", dst.Spec.Source.GitURL)
	}
}

func TestConvertNewToOld_NilBuildPipeline(t *testing.T) {
	src := newapi.Component{
		ObjectMeta: metav1.ObjectMeta{Name: "no-pipeline", Namespace: "ns"},
		Spec: newapi.ComponentSpec{
			Source: newapi.ComponentSource{
				GitURL: "https://github.com/org/repo",
				Versions: []newapi.ComponentVersion{
					{Name: "v1", Revision: "main"},
				},
			},
		},
	}

	dst := ConvertNewToOld(src)

	if dst.Spec.DefaultBuildPipeline != nil {
		t.Error("DefaultBuildPipeline should be nil")
	}
	if dst.Spec.Source.ComponentSourceUnion.Versions[0].BuildPipeline != nil {
		t.Error("Version BuildPipeline should be nil")
	}
}

func TestRoundTrip_NewToOldToNew(t *testing.T) {
	original := newapi.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "roundtrip",
			Namespace: "ns",
		},
		Spec: newapi.ComponentSpec{
			Source: newapi.ComponentSource{
				GitURL:         "https://github.com/org/repo",
				DockerfilePath: "Dockerfile",
				Versions: []newapi.ComponentVersion{
					{
						Name:           "v1",
						Revision:       "main",
						Context:        "src",
						DockerfilePath: "src/Dockerfile",
						SkipBuilds:     true,
					},
				},
			},
			ContainerImage:    "quay.io/org/image",
			SkipOffboardingPr: true,
			RepositorySettings: newapi.RepositorySettings{
				CommentStrategy: "always",
			},
			DefaultBuildPipeline: &newapi.ComponentBuildPipeline{
				PullAndPush: &newapi.PipelineDefinition{
					PipelineRefName: "my-pipeline",
				},
			},
		},
		Status: newapi.ComponentStatus{
			Message:       "status message",
			PacRepository: "pac",
			Versions: []newapi.ComponentVersionStatus{
				{Name: "v1", OnboardingStatus: "succeeded", Revision: "main"},
			},
		},
	}

	old := ConvertNewToOld(original)
	roundtripped := ConvertOldToNew(old)

	if roundtripped.Spec.Source.GitURL != original.Spec.Source.GitURL {
		t.Errorf("GitURL: got %q, want %q", roundtripped.Spec.Source.GitURL, original.Spec.Source.GitURL)
	}
	if roundtripped.Spec.Source.DockerfilePath != original.Spec.Source.DockerfilePath {
		t.Errorf("DockerfilePath: got %q, want %q", roundtripped.Spec.Source.DockerfilePath, original.Spec.Source.DockerfilePath)
	}
	if roundtripped.Spec.ContainerImage != original.Spec.ContainerImage {
		t.Errorf("ContainerImage: got %q, want %q", roundtripped.Spec.ContainerImage, original.Spec.ContainerImage)
	}
	if roundtripped.Spec.SkipOffboardingPr != original.Spec.SkipOffboardingPr {
		t.Error("SkipOffboardingPr mismatch")
	}
	if roundtripped.Spec.RepositorySettings.CommentStrategy != original.Spec.RepositorySettings.CommentStrategy {
		t.Error("RepositorySettings.CommentStrategy mismatch")
	}
	if roundtripped.Spec.DefaultBuildPipeline == nil || roundtripped.Spec.DefaultBuildPipeline.PullAndPush == nil {
		t.Fatal("DefaultBuildPipeline.PullAndPush should survive round-trip")
	}
	if roundtripped.Spec.DefaultBuildPipeline.PullAndPush.PipelineRefName != "my-pipeline" {
		t.Errorf("PipelineRefName: got %q", roundtripped.Spec.DefaultBuildPipeline.PullAndPush.PipelineRefName)
	}
	if len(roundtripped.Spec.Source.Versions) != 1 {
		t.Fatalf("Versions count: got %d", len(roundtripped.Spec.Source.Versions))
	}
	rv := roundtripped.Spec.Source.Versions[0]
	ov := original.Spec.Source.Versions[0]
	if rv.Name != ov.Name || rv.Revision != ov.Revision || rv.Context != ov.Context || rv.DockerfilePath != ov.DockerfilePath || rv.SkipBuilds != ov.SkipBuilds {
		t.Errorf("Version[0] mismatch: got %+v, want %+v", rv, ov)
	}
	if roundtripped.Status.Message != original.Status.Message {
		t.Errorf("Status.Message: got %q, want %q", roundtripped.Status.Message, original.Status.Message)
	}
	if len(roundtripped.Status.Versions) != 1 || roundtripped.Status.Versions[0].Name != "v1" {
		t.Errorf("Status.Versions mismatch: got %+v", roundtripped.Status.Versions)
	}
}

func TestConvertNewToOld_ObjectMetaDeepCopy(t *testing.T) {
	src := newapi.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "copy-test",
			Namespace:   "ns",
			Annotations: map[string]string{"key": "value"},
		},
		Spec: newapi.ComponentSpec{
			Source: newapi.ComponentSource{GitURL: "https://github.com/org/repo"},
		},
	}

	dst := ConvertNewToOld(src)

	// Mutating the destination's annotations should not affect the source
	dst.Annotations["key"] = "mutated"
	if src.Annotations["key"] != "value" {
		t.Error("Mutating dst annotations affected src — ObjectMeta was not deep copied")
	}
}
