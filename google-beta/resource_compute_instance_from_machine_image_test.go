package google

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	compute "google.golang.org/api/compute/v1"
)

func TestAccComputeInstanceFromMachineImage_basic(t *testing.T) {
	t.Parallel()

	var instance compute.Instance
	instanceName := fmt.Sprintf("terraform-test-%s", randString(t, 10))
	generatedInstanceName := fmt.Sprintf("terraform-test-generated-%s", randString(t, 10))
	resourceName := "google_compute_instance_from_machine_image.foobar"

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProvidersOiCS,
		CheckDestroy: testAccCheckComputeInstanceFromMachineImageDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceFromMachineImage_basic(instanceName, generatedInstanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(t, resourceName, &instance),

					// Check that fields were set based on the template
					resource.TestCheckResourceAttr(resourceName, "machine_type", "n1-standard-1"),
					resource.TestCheckResourceAttr(resourceName, "attached_disk.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "scheduling.0.automatic_restart", "false"),
				),
			},
		},
	})
}

func TestAccComputeInstanceFromMachineImage_overrideMetadataDotStartupScript(t *testing.T) {
	t.Parallel()

	var instance compute.Instance
	instanceName := fmt.Sprintf("terraform-test-%s", randString(t, 10))
	generatedInstanceName := fmt.Sprintf("terraform-test-generated-%s", randString(t, 10))
	resourceName := "google_compute_instance_from_machine_image.foobar"

	vcrTest(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProvidersOiCS,
		CheckDestroy: testAccCheckComputeInstanceFromMachineImageDestroyProducer(t),
		Steps: []resource.TestStep{
			{
				Config: testAccComputeInstanceFromMachineImage_overrideMetadataDotStartupScript(instanceName, generatedInstanceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeInstanceExists(t, resourceName, &instance),
					resource.TestCheckResourceAttr(resourceName, "metadata.startup-script", ""),
				),
			},
		},
	})

}

func testAccCheckComputeInstanceFromMachineImageDestroyProducer(t *testing.T) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		config := googleProviderConfig(t)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "google_compute_instance_from_machine_image" {
				continue
			}

			_, err := config.NewComputeClient(config.userAgent).Instances.Get(
				config.Project, rs.Primary.Attributes["zone"], rs.Primary.ID).Do()
			if err == nil {
				return fmt.Errorf("Instance still exists")
			}
		}

		return nil
	}
}

func testAccComputeInstanceFromMachineImage_basic(instance, newInstance string) string {
	return fmt.Sprintf(`
resource "google_compute_instance" "vm" {
  provider     = google-beta

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-10"
    }
  }

  name         = "%s"
  machine_type = "n1-standard-1"

  network_interface {
    network = "default"
  }

  metadata = {
    foo = "bar"
  }

  scheduling {
    automatic_restart = true
  }

  can_ip_forward = true
}

resource "google_compute_machine_image" "foobar" {
  provider        = google-beta
  name            = "%s"
  source_instance = google_compute_instance.vm.self_link
}

resource "google_compute_instance_from_machine_image" "foobar" {
  provider = google-beta
  name = "%s"
  zone = "us-central1-a"

  source_machine_image = google_compute_machine_image.foobar.self_link

  // Overrides
  can_ip_forward = false
  labels = {
    my_key = "my_value"
  }
  scheduling {
    automatic_restart = false
  }
}
`, instance, instance, newInstance)
}

func testAccComputeInstanceFromMachineImage_overrideMetadataDotStartupScript(instanceName, generatedInstanceName string) string {
	return fmt.Sprintf(`
resource "google_compute_instance" "vm" {
  provider     = google-beta

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-10"
    }
  }

  name         = "%s"
  machine_type = "n1-standard-1"

  network_interface {
    network = "default"
  }

  metadata = {
    startup-script = "#!/bin/bash\necho Hello"
  }

}

resource "google_compute_machine_image" "foobar" {
  provider        = google-beta
  name            = "%s"
  source_instance = google_compute_instance.vm.self_link
}

resource "google_compute_instance_from_machine_image" "foobar" {
  provider = google-beta
  name = "%s"
  zone = "us-central1-a"

  source_machine_image = google_compute_machine_image.foobar.self_link

  // Overrides
  metadata = {
    startup-script = ""
  }
}
`, instanceName, instanceName, generatedInstanceName)
}
