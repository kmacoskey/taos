{
  "provider":{
    "google":{
      "project": "data-gp-toolsmiths",
      "region": "us-central1"
    }
  }, 
  "resource":{
    "google_compute_instance":{
      "test":{
        "name": "test",
        "machine_type": "n1-standard-1",
        "zone": "us-central1-a",
        "boot_disk":{
          "initialize_params":{
            "image": "debian-cloud/debian-8"
          }
        },
        "network_interface":{
          "subnetwork": "toolshed-private-subnet"
        }
      }
    }
  },
  "output":{
    "name":{
      "value": "${google_compute_instance.test.name}"
    }
  }
}
