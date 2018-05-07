package reaper_test

import (
	"errors"

	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	log "github.com/sirupsen/logrus"

	"github.com/kmacoskey/taos/models"
	. "github.com/kmacoskey/taos/reaper"
	"github.com/kmacoskey/taos/services"
)

var (
	clusters_map map[string]*models.Cluster
)

var _ = Describe("Reaper", func() {

	var (
		reaper               *ClusterReaper
		valid_interval       string
		valid_request_id     string
		err                  error
		cluster_1            *models.Cluster
		cluster_1_uuid       string
		cluster_2            *models.Cluster
		cluster_2_uuid       string
		invalid_cluster_uuid string
		clusters             []models.Cluster
	)

	BeforeEach(func() {
		log.SetLevel(log.FatalLevel)

		valid_interval = "5s"

		invalid_cluster_uuid = "d1af124a-5141-11e8-9c2d-fa7ae01bbebc"

		valid_request_id = "96bc71ca-518a-11e8-9c2d-fa7ae01bbebc"

		cluster_1_uuid = "a19e2758-0ec5-11e8-ba89-0ed5f89f718b"
		cluster_1 = &models.Cluster{
			Id:              cluster_1_uuid,
			Name:            "cluster",
			Status:          "status",
			TerraformConfig: []byte(`{"provider":{"google":{}}}`),
		}

		cluster_2_uuid = "a19e2bfe-0ec5-11e8-ba89-0ed5f89f718b"
		cluster_2 = &models.Cluster{
			Id:              cluster_2_uuid,
			Name:            "cluster",
			Status:          "status",
			TerraformConfig: []byte(`{"provider":{"google":{}}}`),
		}
	})

	Describe("Reaping clusters", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				clusters_map[cluster_1.Id] = cluster_1
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				err = reaper.ReapClusters()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should reap expired clusters", func() {
				Expect(clusters_map).To(HaveLen(0))
			})
		})

		Context("When there are no clusters to reap", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				err = reaper.ReapClusters()
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Finding Expired Clusters", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				clusters_map[cluster_1.Id] = cluster_1
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				clusters, err = reaper.ExpiredClusters(valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a slice of expired clusters", func() {
				Expect(clusters).To(HaveLen(1))
			})
		})

		Context("When there are no expired clusters", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				clusters, err = reaper.ExpiredClusters(valid_request_id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
			It("Should return a slice of expired clusters", func() {
				Expect(clusters).To(HaveLen(0))
			})
		})
	})

	Describe("Reaping an Expired Cluster", func() {
		Context("When everything goes ok", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				clusters_map[cluster_1.Id] = cluster_1
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				err = reaper.ReapCluster(cluster_1.Id)
			})
			It("Should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("When the cluster does not exist", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				clusters_map[cluster_1.Id] = cluster_1
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				err = reaper.ReapCluster(invalid_cluster_uuid)
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("When no cluster id is given", func() {
			BeforeEach(func() {
				clusters_map = make(map[string]*models.Cluster)
				reaper, err = NewClusterReaper(valid_interval, NewValidClusterService(clusters_map), NewMockDB().db)
				Expect(err).NotTo(HaveOccurred())
				err = reaper.ReapCluster("")
			})
			It("Should error", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

})

func NewMockDB() *MockDB {
	return &MockDB{}
}

type MockDB struct {
	db *sqlx.DB
}

type ValidClusterService struct {
	clustersMap map[string]*models.Cluster
}

func NewValidClusterService(clusters_map map[string]*models.Cluster) *ValidClusterService {
	return &ValidClusterService{
		clustersMap: clusters_map,
	}
}

func (service *ValidClusterService) DeleteCluster(request_id string, client services.TerraformClient, id string) (*models.Cluster, error) {
	if cluster, ok := clusters_map[id]; ok {
		delete(clusters_map, id)
		return cluster, nil
	} else {
		return nil, errors.New("cluster not found")
	}
}

func (service *ValidClusterService) GetExpiredClusters(request_id string) ([]models.Cluster, error) {
	clusters := []models.Cluster{}
	for _, cluster := range clusters_map {
		clusters = append(clusters, *cluster)
	}

	return clusters, nil
}
