/*
Copyright 2017 The Kubernetes Authors All rights reserved.

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

package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/kubernetes/kompose/pkg/kobject"
	"github.com/kubernetes/kompose/pkg/loader/compose"
	"github.com/kubernetes/kompose/pkg/testutils"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	hpa "k8s.io/api/autoscaling/v2beta2"
	api "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
Test the creation of a service
*/
func TestCreateService(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		Environment:   []kobject.EnvVar{{Name: "env", Value: "value"}},
		Port:          []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:       []string{"cmd"},
		WorkingDir:    "dir",
		Args:          []string{"arg1", "arg2"},
		VolList:       []string{"/tmp/volume"},
		Network:       []string{"network1", "network2"}, // not supported
		Labels:        nil,
		Annotations:   map[string]string{"abc": "def"},
		CPUQuota:      1,                    // not supported
		CapAdd:        []string{"cap_add"},  // not supported
		CapDrop:       []string{"cap_drop"}, // not supported
		Expose:        []string{"expose"},   // not supported
		Privileged:    true,
		Restart:       "always",
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	_, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	// Test the creation of the service
	svc := k.CreateService("foo", service)

	if svc.Spec.Ports[0].Port != 123 {
		t.Errorf("Expected port 123 upon conversion, actual %d", svc.Spec.Ports[0].Port)
	}
}

/*
Test the creation of a service with a memory limit and reservation
*/
func TestCreateServiceWithMemLimit(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName:  "name",
		Image:          "image",
		Environment:    []kobject.EnvVar{{Name: "env", Value: "value"}},
		Port:           []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:        []string{"cmd"},
		WorkingDir:     "dir",
		Args:           []string{"arg1", "arg2"},
		VolList:        []string{"/tmp/volume"},
		Network:        []string{"network1", "network2"}, // not supported
		Labels:         nil,
		Annotations:    map[string]string{"abc": "def"},
		CPUQuota:       1,                    // not supported
		CapAdd:         []string{"cap_add"},  // not supported
		CapDrop:        []string{"cap_drop"}, // not supported
		Expose:         []string{"expose"},   // not supported
		Privileged:     true,
		Restart:        "always",
		MemLimit:       1337,
		MemReservation: 1338,
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	// Retrieve the deployment object and test that it matches the mem value
	for _, obj := range objects {
		if deploy, ok := obj.(*appsv1.Deployment); ok {
			memLimit, _ := deploy.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().AsInt64()
			if memLimit != 1337 {
				t.Errorf("Expected 1337 for memory limit check, got %v", memLimit)
			}
			memReservation, _ := deploy.Spec.Template.Spec.Containers[0].Resources.Requests.Memory().AsInt64()
			if memReservation != 1338 {
				t.Errorf("Expected 1338 for memory reservation check, got %v", memReservation)
			}
		}
	}
}

/*
Test the creation of a service with a cpu limit and reservation
*/
func TestCreateServiceWithCPULimit(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName:  "name",
		Image:          "image",
		Environment:    []kobject.EnvVar{{Name: "env", Value: "value"}},
		Port:           []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:        []string{"cmd"},
		WorkingDir:     "dir",
		Args:           []string{"arg1", "arg2"},
		VolList:        []string{"/tmp/volume"},
		Network:        []string{"network1", "network2"}, // not supported
		Labels:         nil,
		Annotations:    map[string]string{"abc": "def"},
		CPUQuota:       1,                    // not supported
		CapAdd:         []string{"cap_add"},  // not supported
		CapDrop:        []string{"cap_drop"}, // not supported
		Expose:         []string{"expose"},   // not supported
		Privileged:     true,
		Restart:        "always",
		CPULimit:       10,
		CPUReservation: 1,
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	// Retrieve the deployment object and test that it matches the cpu value
	for _, obj := range objects {
		if deploy, ok := obj.(*appsv1.Deployment); ok {
			cpuLimit := deploy.Spec.Template.Spec.Containers[0].Resources.Limits.Cpu().MilliValue()
			if cpuLimit != 10 {
				t.Errorf("Expected 10 for cpu limit check, got %v", cpuLimit)
			}
			cpuReservation := deploy.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().MilliValue()
			if cpuReservation != 1 {
				t.Errorf("Expected 1 for cpu reservation check, got %v", cpuReservation)
			}
		}
	}
}

/*
Test the creation of a service with a specified user.
The expected result is that Kompose will set user in PodSpec
*/
func TestCreateServiceWithServiceUser(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		Environment:   []kobject.EnvVar{{Name: "env", Value: "value"}},
		Port:          []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:       []string{"cmd"},
		WorkingDir:    "dir",
		Args:          []string{"arg1", "arg2"},
		VolList:       []string{"/tmp/volume"},
		Network:       []string{"network1", "network2"}, // not supported
		Labels:        nil,
		Annotations:   map[string]string{"kompose.service.type": "nodeport"},
		CPUQuota:      1,                    // not supported
		CapAdd:        []string{"cap_add"},  // not supported
		CapDrop:       []string{"cap_drop"}, // not supported
		Expose:        []string{"expose"},   // not supported
		Privileged:    true,
		Restart:       "always",
		User:          "1234",
	}

	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}

	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 1})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	for _, obj := range objects {
		if deploy, ok := obj.(*appsv1.Deployment); ok {
			uid := *deploy.Spec.Template.Spec.Containers[0].SecurityContext.RunAsUser
			if strconv.FormatInt(uid, 10) != service.User {
				t.Errorf("User in ServiceConfig is not matching user in PodSpec")
			}
		}
	}
}

func TestTransformWithPid(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		Environment:   []kobject.EnvVar{{Name: "env", Value: "value"}},
		Port:          []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:       []string{"cmd"},
		WorkingDir:    "dir",
		Args:          []string{"arg1", "arg2"},
		VolList:       []string{"/tmp/volume"},
		Network:       []string{"network1", "network2"},
		Restart:       "always",
		Pid:           "host",
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	_, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	//for _, obj := range objects {
	//	if deploy, ok := obj.(*appsv1.Deployment); ok {
	//		hostPid := deploy.Spec.Template.Spec.SecurityContext.HostPID
	//		if !hostPid {
	//			t.Errorf("Pid in ServiceConfig is not matching HostPID in PodSpec")
	//		}
	//	}
	//}
}

func TestTransformWithInvalidPid(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		Environment:   []kobject.EnvVar{{Name: "env", Value: "value"}},
		Port:          []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:       []string{"cmd"},
		WorkingDir:    "dir",
		Args:          []string{"arg1", "arg2"},
		VolList:       []string{"/tmp/volume"},
		Network:       []string{"network1", "network2"},
		Restart:       "always",
		Pid:           "badvalue",
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	_, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	//for _, obj := range objects {
	//	if deploy, ok := obj.(*appsv1.Deployment); ok {
	//		if deploy.Spec.Template.Spec.SecurityContext != nil {
	//			hostPid := deploy.Spec.Template.Spec.SecurityContext.HostPID
	//			if hostPid {
	//				t.Errorf("Pid in ServiceConfig is not matching HostPID in PodSpec")
	//			}
	//		}
	//	}
	//}
}

func TestIsDir(t *testing.T) {
	tempPath := "/tmp/kompose_unit"
	tempDir := filepath.Join(tempPath, "i_am_dir")
	tempFile := filepath.Join(tempPath, "i_am_file")
	tempAbsentDirPath := filepath.Join(tempPath, "i_do_not_exist")

	// create directory
	err := os.MkdirAll(tempDir, 0744)
	if err != nil {
		t.Errorf("Unable to create directory: %v", err)
	}

	// create empty file
	f, err := os.Create(tempFile)
	if err != nil {
		t.Errorf("Unable to create empty file: %v", err)
	}
	f.Close()

	// Check output if directory exists
	output, err := isDir(tempDir)
	if err != nil {
		t.Error(errors.Wrap(err, "isDir failed"))
	}
	if !output {
		t.Errorf("directory %v exists but isDir() returned %v", tempDir, output)
	}

	// Check output if file is provided
	output, err = isDir(tempFile)
	if err != nil {
		t.Error(errors.Wrap(err, "isDir failed"))
	}
	if output {
		t.Errorf("%v is a file but isDir() returned %v", tempDir, output)
	}

	// Check output if path does not exist
	output, err = isDir(tempAbsentDirPath)
	if err != nil {
		t.Error(errors.Wrap(err, "isDir failed"))
	}
	if output {
		t.Errorf("Directory %v does not exist, but isDir() returned %v", tempAbsentDirPath, output)
	}

	// delete temporary directory
	err = os.RemoveAll(tempPath)
	if err != nil {
		t.Errorf("Error removing the temporary directory during cleanup: %v", err)
	}
}

// TestServiceWithHealthCheck this tests if Headless Service is created for services with HealthCheck.
func TestServiceWithHealthCheck(t *testing.T) {
	testCases := map[string]struct {
		service kobject.ServiceConfig
	}{
		"Exec": {
			service: kobject.ServiceConfig{
				ContainerName: "name",
				Image:         "image",
				ServiceType:   "Headless",
				HealthChecks: kobject.HealthChecks{
					Readiness: kobject.HealthCheck{
						Test:        []string{"arg1", "arg2"},
						Timeout:     10,
						Interval:    5,
						Retries:     3,
						StartPeriod: 60,
					},
					Liveness: kobject.HealthCheck{
						Test:        []string{"arg1", "arg2"},
						Timeout:     11,
						Interval:    6,
						Retries:     4,
						StartPeriod: 61,
					},
				},
			},
		},
		"HTTPGet": {
			service: kobject.ServiceConfig{
				ContainerName: "name",
				Image:         "image",
				ServiceType:   "Headless",
				HealthChecks: kobject.HealthChecks{
					Readiness: kobject.HealthCheck{
						HTTPPath:    "/health",
						HTTPPort:    8080,
						Timeout:     10,
						Interval:    5,
						Retries:     3,
						StartPeriod: 60,
					},
					Liveness: kobject.HealthCheck{
						HTTPPath:    "/ready",
						HTTPPort:    8080,
						Timeout:     11,
						Interval:    6,
						Retries:     4,
						StartPeriod: 61,
					},
				},
			},
		},
		"TCPSocket": {
			service: kobject.ServiceConfig{
				ContainerName: "name",
				Image:         "image",
				ServiceType:   "Headless",
				HealthChecks: kobject.HealthChecks{
					Readiness: kobject.HealthCheck{
						TCPPort:     8080,
						Timeout:     10,
						Interval:    5,
						Retries:     3,
						StartPeriod: 60,
					},
					Liveness: kobject.HealthCheck{
						TCPPort:     8080,
						Timeout:     11,
						Interval:    6,
						Retries:     4,
						StartPeriod: 61,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		k := Kubernetes{}
		komposeObject := kobject.KomposeObject{
			ServiceConfigs: map[string]kobject.ServiceConfig{"app": testCase.service},
		}
		objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 1})
		if err != nil {
			t.Error(errors.Wrap(err, "k.Transform failed"))
		}
		if err := testutils.CheckForHealthCheckLivenessAndReadiness(objects); err != nil {
			t.Error(err)
		}
	}
}

// TestServiceWithoutPort this tests if Headless Service is created for services without Port.
func TestServiceWithoutPort(t *testing.T) {
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		ServiceType:   "Headless",
	}

	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}

	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 1})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}
	if err := testutils.CheckForHeadless(objects); err != nil {
		t.Error(err)
	}
}

// Tests if deployment strategy is being set to Recreate when volumes are
// present
func TestRecreateStrategyWithVolumesPresent(t *testing.T) {
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		VolList:       []string{"/tmp/volume"},
		Volumes:       []kobject.Volumes{{SvcName: "app", MountPath: "/tmp/volume", PVCName: "app-claim0"}},
	}

	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}

	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 1})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}
	for _, obj := range objects {
		if deployment, ok := obj.(*appsv1.Deployment); ok {
			if deployment.Spec.Strategy.Type != appsv1.RecreateDeploymentStrategyType {
				t.Errorf("Expected %v as Strategy Type, got %v",
					appsv1.RecreateDeploymentStrategyType,
					deployment.Spec.Strategy.Type)
			}
		}
	}
}

func TestSortedKeys(t *testing.T) {
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
	}
	service1 := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
	}
	c := []string{"a", "b"}

	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"b": service, "a": service1},
	}
	a := SortedKeys(komposeObject.ServiceConfigs)
	if !reflect.DeepEqual(a, c) {
		t.Logf("Test Fail output should be %s", c)
	}
}

// test conversion from duration string to seconds *int64
func TestDurationStrToSecondsInt(t *testing.T) {
	testCases := map[string]struct {
		in  string
		out *int64
	}{
		"5s":         {in: "5s", out: &[]int64{5}[0]},
		"1m30s":      {in: "1m30s", out: &[]int64{90}[0]},
		"empty":      {in: "", out: nil},
		"onlynumber": {in: "2", out: nil},
		"illegal":    {in: "abc", out: nil},
	}

	for name, test := range testCases {
		result, _ := DurationStrToSecondsInt(test.in)
		if test.out == nil && result != nil {
			t.Errorf("Case '%v' for TestDurationStrToSecondsInt fail, Expected 'nil' , got '%v'", name, *result)
		}
		if test.out != nil && result == nil {
			t.Errorf("Case '%v' for TestDurationStrToSecondsInt fail, Expected '%v' , got 'nil'", name, *test.out)
		}
		if test.out != nil && result != nil && *test.out != *result {
			t.Errorf("Case '%v' for TestDurationStrToSecondsInt fail, Expected '%v' , got '%v'", name, *test.out, *result)
		}
	}
}

func TestServiceWithServiceAccount(t *testing.T) {
	assertServiceAccountName := "my-service"

	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		Port:          []kobject.Ports{{HostPort: 55555}},
		Labels:        map[string]string{compose.LabelServiceAccountName: assertServiceAccountName},
	}

	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}

	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}
	for _, obj := range objects {
		if deployment, ok := obj.(*appsv1.Deployment); ok {
			if deployment.Spec.Template.Spec.ServiceAccountName != assertServiceAccountName {
				t.Errorf("Expected %v returned, got %v", assertServiceAccountName, deployment.Spec.Template.Spec.ServiceAccountName)
			}
		}
	}
}

func TestCreateServiceWithSpecialName(t *testing.T) {
	service := kobject.ServiceConfig{
		ContainerName: "front_end",
		Image:         "nginx",
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}
	expectedContainerName := "front-end"
	for _, obj := range objects {
		if deploy, ok := obj.(*appsv1.Deployment); ok {
			containerName := deploy.Spec.Template.Spec.Containers[0].Name
			if containerName != "front-end" {
				t.Errorf("Error while transforming container name. Expected %s Got %s", expectedContainerName, containerName)
			}
		}
	}
}

func TestArgsInterpolation(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		Environment:   []kobject.EnvVar{{Name: "PROTOCOL", Value: "https"}, {Name: "DOMAIN", Value: "google.com"}},
		Port:          []kobject.Ports{{HostPort: 123, ContainerPort: 456, Protocol: string(corev1.ProtocolTCP)}},
		Command:       []string{"curl"},
		Args:          []string{"$PROTOCOL://$DOMAIN/"},
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true, Replicas: 3})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	expectedArgs := []string{"$(PROTOCOL)://$(DOMAIN)/"}
	for _, obj := range objects {
		if deployment, ok := obj.(*appsv1.Deployment); ok {
			args := deployment.Spec.Template.Spec.Containers[0].Args[0]
			if args != expectedArgs[0] {
				t.Errorf("Expected args %v upon conversion, actual %v", expectedArgs, args)
			}
		}
	}
}

func TestReadOnlyRootFS(t *testing.T) {
	// An example service
	service := kobject.ServiceConfig{
		ContainerName: "name",
		Image:         "image",
		ReadOnly:      true,
	}

	// An example object generated via k8s runtime.Objects()
	komposeObject := kobject.KomposeObject{
		ServiceConfigs: map[string]kobject.ServiceConfig{"app": service},
	}
	k := Kubernetes{}
	objects, err := k.Transform(komposeObject, kobject.ConvertOptions{CreateD: true})
	if err != nil {
		t.Error(errors.Wrap(err, "k.Transform failed"))
	}

	for _, obj := range objects {
		if deployment, ok := obj.(*appsv1.Deployment); ok {
			readOnlyFS := deployment.Spec.Template.Spec.Containers[0].SecurityContext.ReadOnlyRootFilesystem
			if *readOnlyFS != true {
				t.Errorf("Expected ReadOnlyRootFileSystem %v upon conversion, actual %v", true, readOnlyFS)
			}
		}
	}
}

func TestFormatEnvName(t *testing.T) {
	type args struct {
		name        string
		serviceName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "check dot conversion",
			args: args{
				name: "random.test",
			},
			want: "random-test",
		},
		{
			name: "check that path is shortened",
			args: args{
				name: "random/test/v1",
			},
			want: "v1",
		},
		{
			name: "check that ./ is removed",
			args: args{
				name: "./random",
			},
			want: "random",
		},
		{
			name: "check that ./ is removed",
			args: args{
				name: "abcdefghijklnmopqrstuvxyzabcdefghijklmnopqrstuvwxyzabcdejghijkl$Hereisadditional",
			},
			want: "abcdefghijklnmopqrstuvxyzabcdefghijklmnopqrstuvwxyzabcdejghijkl",
		},
		{
			name: "check that not begins with -",
			args: args{
				name:        "src/app/.env",
				serviceName: "app",
			},
			want: "app-env",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatEnvName(tt.args.name, tt.args.serviceName); got != tt.want {
				t.Errorf("FormatEnvName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test empty interfaces removal
func TestRemoveEmptyInterfaces(t *testing.T) {
	type Obj = map[string]interface{}
	var testCases = []struct {
		input  interface{}
		output interface{}
	}{
		{Obj{"useless": Obj{}}, Obj{}},
		{Obj{"usefull": Obj{"usefull": "usefull"}}, Obj{"usefull": Obj{"usefull": "usefull"}}},
		{Obj{"usefull": Obj{"usefull": "usefull", "uselessdeep": Obj{}, "uselessnil": nil}}, Obj{"usefull": Obj{"usefull": "usefull"}}},
		{Obj{"uselessdeep": Obj{"uselessdeep": Obj{}, "uselessnil": nil}}, Obj{}},
		{Obj{"uselessempty": []interface{}{nil}}, Obj{}},
		{"test", "test"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Test removeEmptyInterfaces(%s)", tc.input), func(t *testing.T) {
			result := removeEmptyInterfaces(tc.input)
			if !reflect.DeepEqual(result, tc.output) {
				t.Errorf("Expected %v, got %v", tc.output, result)
			}
		})
	}
}

func Test_parseContainerCommandsFromStr(t *testing.T) {
	tests := []struct {
		name string
		line string
		want []string
	}{
		{
			name: "line command without spaces in between",
			line: `[ "bundle", "exec", "thin", "-p", "3000" ]`,
			want: []string{
				"bundle", "exec", "thin", "-p", "3000",
			},
		},
		{
			name: `line command spaces inside ""`,
			line: `[ " bundle ",   " exec ", " thin ", " -p ", "3000" ]`,
			want: []string{
				"bundle", "exec", "thin", "-p", "3000",
			},
		},
		{
			name: `more use cases for line command spaces inside ""`,
			line: `[  " bundle ",   "exec ",   " thin ", " -p ", "3000  " ]`,
			want: []string{
				"bundle", "exec", "thin", "-p", "3000",
			},
		},
		{
			name: `line command without [] and ""`,
			line: `bundle exec thin -p 3000`,
			want: []string{
				"bundle exec thin -p 3000",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseContainerCommandsFromStr(tt.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseContainerCommandsFromStr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fillInitContainers(t *testing.T) {
	type args struct {
		template *api.PodTemplateSpec
		service  kobject.ServiceConfig
	}
	tests := []struct {
		name string
		args args
		want []corev1.Container
	}{
		{
			name: "Testing init container are generated from labels with ,",
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:    "name",
						compose.LabelInitContainerImage:   "image",
						compose.LabelInitContainerCommand: `[ "bundle", "exec", "thin", "-p", "3000" ]`,
					},
				},
			},
			want: []corev1.Container{
				{
					Name:  "name",
					Image: "image",
					Command: []string{
						"bundle", "exec", "thin", "-p", "3000",
					},
				},
			},
		},
		{
			name: "Testing init container are generated from labels without ,",
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:    "name",
						compose.LabelInitContainerImage:   "image",
						compose.LabelInitContainerCommand: `bundle exec thin -p 3000`,
					},
				},
			},
			want: []corev1.Container{
				{
					Name:  "name",
					Image: "image",
					Command: []string{
						`bundle exec thin -p 3000`,
					},
				},
			},
		},
		{
			name: `Testing init container with long command with vars inside and ''`,
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:    "init-myservice",
						compose.LabelInitContainerImage:   "busybox:1.28",
						compose.LabelInitContainerCommand: `['sh', '-c', "until nslookup myservice.$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace).svc.cluster.local; do echo waiting for myservice; sleep 2; done"]`,
					},
				},
			},
			want: []corev1.Container{
				{
					Name:  "init-myservice",
					Image: "busybox:1.28",
					Command: []string{
						"sh", "-c", `until nslookup myservice.$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace).svc.cluster.local; do echo waiting for myservice; sleep 2; done`,
					},
				},
			},
		},
		{
			name: `without image`,
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:    "init-myservice",
						compose.LabelInitContainerImage:   "",
						compose.LabelInitContainerCommand: `['sh', '-c', "until nslookup myservice.$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace).svc.cluster.local; do echo waiting for myservice; sleep 2; done"]`,
					},
				},
			},
			want: nil,
		},
		{
			name: `Testing init container without name`,
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:    "",
						compose.LabelInitContainerImage:   "busybox:1.28",
						compose.LabelInitContainerCommand: `['sh', '-c', "until nslookup myservice.$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace).svc.cluster.local; do echo waiting for myservice; sleep 2; done"]`,
					},
				},
			},
			want: []corev1.Container{
				{
					Name:  "init-service",
					Image: "busybox:1.28",
					Command: []string{
						"sh", "-c", `until nslookup myservice.$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace).svc.cluster.local; do echo waiting for myservice; sleep 2; done`,
					},
				},
			},
		},
		{
			name: `Testing init container without command`,
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:    "init-service",
						compose.LabelInitContainerImage:   "busybox:1.28",
						compose.LabelInitContainerCommand: ``,
					},
				},
			},
			want: []corev1.Container{
				{
					Name:    "init-service",
					Image:   "busybox:1.28",
					Command: []string{},
				},
			},
		},
		{
			name: `Testing init container without command`,
			args: args{
				template: &api.PodTemplateSpec{},
				service: kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelInitContainerName:  "init-service",
						compose.LabelInitContainerImage: "busybox:1.28",
					},
				},
			},
			want: []corev1.Container{
				{
					Name:    "init-service",
					Image:   "busybox:1.28",
					Command: []string{},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fillInitContainers(tt.args.template, tt.args.service)
			if !reflect.DeepEqual(tt.args.template.Spec.InitContainers, tt.want) {
				t.Errorf("Test_fillInitContainers Fail got %v, want %v", tt.args.template.Spec.InitContainers, tt.want)
			}
		})
	}
}

func Test_getHpaValue(t *testing.T) {
	type args struct {
		service      *kobject.ServiceConfig
		label        string
		defaultValue int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		// LabelHpaMinReplicas
		{
			name: "LabelHpaMinReplicas with 1 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMinReplicas,
				defaultValue: 1,
			},
			want: 1,
		},
		{
			name: "LabelHpaMinReplicas with 0 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "0",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMinReplicas,
				defaultValue: 1,
			},
			want: 0,
		},
		{
			name: "LabelHpaMinReplicas with error value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "cannot transform",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMinReplicas,
				defaultValue: 1,
			},
			want: 1,
		},
		// LabelHpaMaxReplicas
		{
			name: "LabelHpaMaxReplicas with 10 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMaxReplicas,
				defaultValue: 30,
			},
			want: 10,
		},
		{
			name: "LabelHpaMaxReplicas with 0 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "0",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMaxReplicas,
				defaultValue: DefaultMaxReplicas,
			},
			want: 0,
		},
		{
			name: "LabelHpaMaxReplicas with error value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "cannot transform",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMaxReplicas,
				defaultValue: DefaultMaxReplicas,
			},
			want: DefaultMaxReplicas,
		},
		// LabelHpaCPU
		{
			name: "LabelHpaCPU with 50 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaCPU,
				defaultValue: 30,
			},
			want: 50,
		},
		{
			name: "LabelHpaCPU with 0 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "0",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: 0,
		},
		{
			name: "LabelHpaCPU with error value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "cannot transform",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: DefaultCPUUtilization,
		},
		// LabelHpaMemory
		{
			name: "LabelHpaMemory with 70 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
				label:        compose.LabelHpaMemory,
				defaultValue: 30,
			},
			want: 70,
		},
		{
			name: "LabelHpaMemory with 0 value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "0",
					},
				},
				label:        compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: 0,
		},
		{
			name: "LabelHpaMemory with error value",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "cannot transform",
					},
				},
				label:        compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: DefaultMemoryUtilization,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHpaValue(tt.args.service, tt.args.label, tt.args.defaultValue); got != tt.want {
				t.Errorf("getHpaValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getResourceHpaValues(t *testing.T) {
	type args struct {
		service *kobject.ServiceConfig
	}
	tests := []struct {
		name string
		args args
		want HpaValues
	}{
		{
			name: "check default values",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "3",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       1,
				MaxReplicas:       3,
				CPUtilization:     50,
				MemoryUtilization: 70,
			},
		},
		{
			name: "check if max replicas are less than min replicas, and max replicas set to min replicas",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "5",
						compose.LabelHpaMaxReplicas: "3",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       5,
				MaxReplicas:       5, // same as min replicas
				CPUtilization:     50,
				MemoryUtilization: 70,
			},
		},
		{
			name: "with error values and use default values from LabelHpaMinReplicas",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "cannot transform",
						compose.LabelHpaMaxReplicas: "3",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       3,
				CPUtilization:     50,
				MemoryUtilization: 70,
			},
		},
		{
			name: "LabelHpaMaxReplicas is minor to LabelHpaMinReplicas",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "6",
						compose.LabelHpaMaxReplicas: "5",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       6,
				MaxReplicas:       6, // set min replicas number
				CPUtilization:     50,
				MemoryUtilization: 70,
			},
		},
		{
			name: "error label and LabelHpaMaxReplicas is minor to LabelHpaMinReplicas",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "6",
						compose.LabelHpaMaxReplicas: "5",
						compose.LabelHpaCPU:         "cannot transform",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       6,
				MaxReplicas:       6, // same as min replicas number
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: 70,
			},
		},
		{
			name: "error label and LabelHpaMaxReplicas is minor to LabelHpaMinReplicas and cannot transform hpa mmemor utilization",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "6",
						compose.LabelHpaMaxReplicas: "5",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "cannot transform",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       6,
				MaxReplicas:       6,
				CPUtilization:     50,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "all error label, set all default values",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "cannot transform",
						compose.LabelHpaMaxReplicas: "cannot transform",
						compose.LabelHpaCPU:         "cannot transform",
						compose.LabelHpaMemory:      "cannot transform",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "error label without some labels, missing labels set to default",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "cannot transform",
						compose.LabelHpaMaxReplicas: "cannot transform",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "without labels, should return default values",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "only min replicas label is provided",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "3",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       3,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "only max replicas label is provided",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMaxReplicas: "5",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       5,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "check default values when all labels contain invalid values",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "cannot transform",
						compose.LabelHpaMaxReplicas: "cannot transform",
						compose.LabelHpaCPU:         "cannot transform",
						compose.LabelHpaMemory:      "cannot transform",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "only cpu utilization label is provided",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "80",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     80,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "only memory utilization label is provided",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "90",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: 90,
			},
		},
		{
			name: "only cpu and memory utilization labels are provided",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU:    "80",
						compose.LabelHpaMemory: "90",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     80,
				MemoryUtilization: 90,
			},
		},
		{
			name: "check default values when labels are empty strings",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "",
						compose.LabelHpaMaxReplicas: "",
						compose.LabelHpaCPU:         "",
						compose.LabelHpaMemory:      "",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "check default values when labels contain invalid characters",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "abc",
						compose.LabelHpaMaxReplicas: "xyz",
						compose.LabelHpaCPU:         "-100",
						compose.LabelHpaMemory:      "invalid",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "check default values when labels are set to zero",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "0",
						compose.LabelHpaMaxReplicas: "0",
						compose.LabelHpaCPU:         "0",
						compose.LabelHpaMemory:      "0",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       0,
				MaxReplicas:       0,
				CPUtilization:     50,
				MemoryUtilization: 70,
			},
		},
		{
			name: "check default values when all labels are negative",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "-5",
						compose.LabelHpaMaxReplicas: "-10",
						compose.LabelHpaCPU:         "-20",
						compose.LabelHpaMemory:      "-30",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
		{
			name: "check default values when labels cpu and memory are over",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "-2",
						compose.LabelHpaMaxReplicas: "-2",
						compose.LabelHpaCPU:         "120",
						compose.LabelHpaMemory:      "120",
					},
				},
			},
			want: HpaValues{
				MinReplicas:       DefaultMinReplicas,
				MaxReplicas:       DefaultMaxReplicas,
				CPUtilization:     DefaultCPUUtilization,
				MemoryUtilization: DefaultMemoryUtilization,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getResourceHpaValues(tt.args.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getResourceHpaValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validatePercentageMetric(t *testing.T) {
	type args struct {
		service      *kobject.ServiceConfig
		metricLabel  string
		defaultValue int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "0 cpu utilization",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "0",
					},
				},
				metricLabel:  compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: 50,
		},
		{
			name: "default cpu valid range",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "120",
					},
				},
				metricLabel:  compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: DefaultCPUUtilization,
		},
		{
			name: "cpu invalid range",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "-120",
					},
				},
				metricLabel:  compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: DefaultCPUUtilization,
		},
		{
			name: "cpu utilization set to 100",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "100",
					},
				},
				metricLabel:  compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: 100,
		},
		{
			name: "cpu utlization set to 101",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "101",
					},
				},
				metricLabel:  compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: DefaultCPUUtilization,
		},
		{
			name: "cannot convert value in cpu label",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaCPU: "not converted",
					},
				},
				metricLabel:  compose.LabelHpaCPU,
				defaultValue: DefaultCPUUtilization,
			},
			want: DefaultCPUUtilization,
		},
		{
			name: "0 memory utilization",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "0",
					},
				},
				metricLabel:  compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: 70,
		},
		{
			name: "memory over 100 utilization",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "120",
					},
				},
				metricLabel:  compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: DefaultMemoryUtilization,
		},
		{
			name: "-120 utilization memory wrong range",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "-120",
					},
				},
				metricLabel:  compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: DefaultMemoryUtilization,
		},
		{
			name: "memory 100 usage",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "100",
					},
				},
				metricLabel:  compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: 100,
		},
		{
			name: "101 memory utilization",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "101",
					},
				},
				metricLabel:  compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: DefaultMemoryUtilization,
		},
		{
			name: "cannot convert memory from label",
			args: args{
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMemory: "not converted",
					},
				},
				metricLabel:  compose.LabelHpaMemory,
				defaultValue: DefaultMemoryUtilization,
			},
			want: DefaultMemoryUtilization,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validatePercentageMetric(tt.args.service, tt.args.metricLabel, tt.args.defaultValue); got != tt.want {
				t.Errorf("validatePercentageMetric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getHpaMetricSpec(t *testing.T) {
	valueCPUFixed := int32(50)
	valueMemoryFixed := int32(70)
	valueOver100 := int32(120)
	valueUnderZero := int32(-120)
	// valueZero := int32(0)
	type args struct {
		hpaValues HpaValues
	}
	tests := []struct {
		name string
		args args
		want []hpa.MetricSpec
	}{
		{
			name: "no values",
			args: args{
				hpaValues: HpaValues{}, // set all values to 0
			},
			want: nil,
		},
		{
			name: "only cpu",
			args: args{
				hpaValues: HpaValues{
					CPUtilization: valueCPUFixed,
				},
			},
			want: []hpa.MetricSpec{
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "cpu",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueCPUFixed,
						},
					},
				},
			},
		},
		{
			name: "only memory",
			args: args{
				hpaValues: HpaValues{
					MemoryUtilization: 70,
				},
			},
			want: []hpa.MetricSpec{
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "memory",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueMemoryFixed,
						},
					},
				},
			},
		},
		{
			name: "cpu and memory",
			args: args{
				hpaValues: HpaValues{
					CPUtilization:     valueCPUFixed,
					MemoryUtilization: valueMemoryFixed,
				},
			},
			want: []hpa.MetricSpec{
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "cpu",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueCPUFixed,
						},
					},
				},
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "memory",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueMemoryFixed,
						},
					},
				},
			},
		},
		{
			name: "memory over 100",
			args: args{
				hpaValues: HpaValues{
					MemoryUtilization: valueOver100,
				},
			},
			want: []hpa.MetricSpec{
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "memory",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueOver100,
						},
					},
				},
			},
		},
		{
			name: "cpu and memory over 100",
			args: args{
				hpaValues: HpaValues{
					CPUtilization:     valueOver100,
					MemoryUtilization: valueOver100,
				},
			},
			want: []hpa.MetricSpec{
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "cpu",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueOver100,
						},
					},
				},
				{
					Type: hpa.ResourceMetricSourceType,
					Resource: &hpa.ResourceMetricSource{
						Name: "memory",
						Target: hpa.MetricTarget{
							Type:               hpa.UtilizationMetricType,
							AverageUtilization: &valueOver100,
						},
					},
				},
			},
		},
		{
			name: "cpu and memory under 0",
			args: args{
				hpaValues: HpaValues{
					CPUtilization:     valueUnderZero,
					MemoryUtilization: valueUnderZero,
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHpaMetricSpec(tt.args.hpaValues); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getHpaMetricSpec() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createHPAResources(t *testing.T) {
	valueCPUFixed := int32(50)
	valueMemoryFixed := int32(70)
	fixedMinReplicas := int32(1)
	type args struct {
		name    string
		service *kobject.ServiceConfig
	}
	tests := []struct {
		name string
		args args
		want hpa.HorizontalPodAutoscaler
	}{
		{
			name: "all labels",
			args: args{
				name: "web",
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "10",
						compose.LabelHpaCPU:         "50",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: hpa.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "web",
				},
				Spec: hpa.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: hpa.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "web",
						APIVersion: "apps/v1",
					},
					MinReplicas: &fixedMinReplicas,
					MaxReplicas: 10,
					Metrics: []hpa.MetricSpec{
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "cpu",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueCPUFixed,
								},
							},
						},
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "memory",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueMemoryFixed,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "minimum labels",
			args: args{
				name: "api",
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaCPU:         "50",
					},
				},
			},
			want: hpa.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "api",
				},
				Spec: hpa.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: hpa.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "api",
						APIVersion: "apps/v1",
					},
					MinReplicas: &fixedMinReplicas,
					MaxReplicas: DefaultMaxReplicas,
					Metrics: []hpa.MetricSpec{
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "cpu",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueCPUFixed,
								},
							},
						},
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "memory",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueMemoryFixed,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "missing CPU utilization label",
			args: args{
				name: "app",
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "5",
						compose.LabelHpaMemory:      "70",
					},
				},
			},
			want: hpa.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "app",
				},
				Spec: hpa.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: hpa.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "app",
						APIVersion: "apps/v1",
					},
					MinReplicas: &fixedMinReplicas,
					MaxReplicas: 5,
					Metrics: []hpa.MetricSpec{
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "cpu",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueCPUFixed,
								},
							},
						},
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "memory",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueMemoryFixed,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "missing memory utilization label",
			args: args{
				name: "db",
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "8",
						compose.LabelHpaCPU:         "50",
					},
				},
			},
			want: hpa.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "db",
				},
				Spec: hpa.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: hpa.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "db",
						APIVersion: "apps/v1",
					},
					MinReplicas: &fixedMinReplicas,
					MaxReplicas: 8,
					Metrics: []hpa.MetricSpec{
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "cpu",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueCPUFixed,
								},
							},
						},
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "memory",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueMemoryFixed,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "wrong labels",
			args: args{
				name: "db",
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "not converted",
						compose.LabelHpaMaxReplicas: "not converted",
					},
				},
			},
			want: hpa.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "db",
				},
				Spec: hpa.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: hpa.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "db",
						APIVersion: "apps/v1",
					},
					MinReplicas: &fixedMinReplicas,
					MaxReplicas: DefaultMaxReplicas,
					Metrics: []hpa.MetricSpec{
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "cpu",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueCPUFixed,
								},
							},
						},
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "memory",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueMemoryFixed,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "missing both CPU and memory utilization labels",
			args: args{
				name: "db",
				service: &kobject.ServiceConfig{
					Labels: map[string]string{
						compose.LabelHpaMinReplicas: "1",
						compose.LabelHpaMaxReplicas: "5",
					},
				},
			},
			want: hpa.HorizontalPodAutoscaler{
				TypeMeta: metav1.TypeMeta{
					Kind:       "HorizontalPodAutoscaler",
					APIVersion: "autoscaling/v2",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "db",
				},
				Spec: hpa.HorizontalPodAutoscalerSpec{
					ScaleTargetRef: hpa.CrossVersionObjectReference{
						Kind:       "Deployment",
						Name:       "db",
						APIVersion: "apps/v1",
					},
					MinReplicas: &fixedMinReplicas,
					MaxReplicas: 5,
					Metrics: []hpa.MetricSpec{
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "cpu",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueCPUFixed,
								},
							},
						},
						{
							Type: hpa.ResourceMetricSourceType,
							Resource: &hpa.ResourceMetricSource{
								Name: "memory",
								Target: hpa.MetricTarget{
									Type:               hpa.UtilizationMetricType,
									AverageUtilization: &valueMemoryFixed,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createHPAResources(tt.args.name, tt.args.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createHPAResources() = %v, want %v", got, tt.want)
			}
		})
	}
}
