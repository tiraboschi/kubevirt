/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package tests_test

import (
	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/tests/console"
)

var _ = Describe("[sig-compute]vTPM", decorators.SigCompute, func() {

	Context("[rfe_id:5168][crit:high][vendor:cnv-qe@redhat.com][level:component] with TPM VMI option enabled", func() {
		It("[test_id:8607] should expose a functional emulated TPM which persists across migrations", decorators.RequiresTwoSchedulableNodes, decorators.Conformance, func() {
			By("Creating a VMI with TPM enabled")
			vmi := libvmifact.NewFedora(
				libnet.WithMasqueradeNetworking(),
				libvmi.WithTPM(false),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)

			By("Logging in")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Ensuring a TPM device is present")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /dev/tpm*\n"},
				&expect.BExp{R: "/dev/tpm0"},
			}, 300)).To(Succeed(), "Could not find a TPM device")

			const expectedState = "0x1EE66777C372B96BC74AC4CB892E0879FA3CCF6A2F53DB1D00FD18B264797F49"
			By("Ensuring the TPM device is functional")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_pcrread sha256:15\n"},
				&expect.BExp{R: "0x0000000000000000000000000000000000000000000000000000000000000000"},
				&expect.BSnd{S: "tpm2_pcrextend 15:sha256=54d626e08c1c802b305dad30b7e54a82f102390cc92c7d4db112048935236e9c && echo 'do''ne'\n"},
				&expect.BExp{R: "done"},
				&expect.BSnd{S: "tpm2_pcrread sha256:15\n"},
				&expect.BExp{R: expectedState},
			}, 300)).To(Succeed(), "PCR extension doesn't work correctly")

			By("Migrating the VMI")
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(),
				libmigration.New(vmi.Name, vmi.Namespace))

			By("Ensuring the TPM is still functional and its state carried over")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "tpm2_pcrread sha256:15\n"},
				&expect.BExp{R: expectedState},
			}, 300)).To(Succeed(), "Migrating broke the TPM")
		})
	})
})
