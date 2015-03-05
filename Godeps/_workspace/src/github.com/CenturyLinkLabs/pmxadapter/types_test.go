package pmxadapter

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshalVolume(t *testing.T) {

	volume := Volume{HostPath: "foo", ContainerPath: "bar"}
	jsonTest, _ := json.Marshal(volume)

	assert.Equal(t, `{"hostPath":"foo","containerPath":"bar"}`, string(jsonTest))

}

func TestUnmarshalVolume(t *testing.T) {

	volume := &Volume{}
	str := `{"hostPath": "one", "containerPath": "two"}`
	json.Unmarshal([]byte(str), &volume)

	assert.Equal(t, "one", volume.HostPath)
	assert.Equal(t, "two", volume.ContainerPath)
}

func TestMarshalEnvironment(t *testing.T) {

	environment := Environment{Variable: "start", Value: "end"}
	jsonTest, _ := json.Marshal(environment)

	assert.Equal(t, `{"variable":"start","value":"end"}`, string(jsonTest))
}

func TestUnmarshalEnvironment(t *testing.T) {

	environment := &Environment{}
	str := `{"variable": "me", "value": "you"}`
	json.Unmarshal([]byte(str), &environment)

	assert.Equal(t, "me", environment.Variable)
	assert.Equal(t, "you", environment.Value)
}

func TestMarshalPort(t *testing.T) {
	port := Port{HostPort: 9000, ContainerPort: 0, Protocol: "TCP"}
	jsonTest, _ := json.Marshal(port)

	assert.Equal(t, `{"hostPort":9000,"containerPort":0,"protocol":"TCP"}`, string(jsonTest))
}

func TestUnmarshalPort(t *testing.T) {

	port := &Port{}
	str := `{"hostPort": 8080, "containerPort": 9000}`
	json.Unmarshal([]byte(str), &port)

	assert.Equal(t, 8080, int(port.HostPort))
	assert.Equal(t, 9000, int(port.ContainerPort))
	assert.Equal(t, "", port.Protocol)
}

func TestMarshalLink(t *testing.T) {

	link := Link{Name: "start", Alias: "end"}
	jsonTest, _ := json.Marshal(link)

	assert.Equal(t, `{"name":"start","alias":"end"}`, string(jsonTest))
}

func TestUnmarshalLink(t *testing.T) {

	link := &Link{}
	str := `{"name": "me", "alias": "you"}`
	json.Unmarshal([]byte(str), &link)

	assert.Equal(t, "me", link.Name)
	assert.Equal(t, "you", link.Alias)
}

func TestMarshalService(t *testing.T) {

	link := Link{Name: "db", Alias: "db_1"}
	port := Port{HostPort: 8080, ContainerPort: 8080}
	environment := Environment{Variable: "start", Value: "end"}
	volume := Volume{HostPath: "foo", ContainerPath: "bar"}
	volumesFrom := VolumesFrom{Name: "myvolume"}
	service := Service{Name: "myServiceName", Source: "centurylink/service", Command: "/run.sh",
		Links:       []*Link{&link},
		Ports:       []*Port{&port},
		Environment: []*Environment{&environment},
		Volumes:     []*Volume{&volume},
		VolumesFrom: []*VolumesFrom{&volumesFrom},
		Expose:      []uint16{8080, 9000}}

	jsonTest, _ := json.Marshal(service)

	assert.Equal(t, `{"name":"myServiceName","source":"centurylink/service","command":"/run.sh","links":[{"name":"db","alias":"db_1"}],"ports":[{"hostPort":8080,"containerPort":8080}],"expose":[8080,9000],"environment":[{"variable":"start","value":"end"}],"volumes":[{"hostPath":"foo","containerPath":"bar"}],"volumes_from":[{"name":"myvolume"}],"deployment":{}}`, string(jsonTest))
}

func TestUnmarshalService(t *testing.T) {

	service := &Service{}
	str := `{"name":"myServiceName","source":"centurylink/service","command":"/run.sh","links":[{"name":"db","alias":"db_1"}],"ports":[{"hostPort":8080,"containerPort":8080}],"expose":[8080,9000],"environment":[{"variable":"start","value":"end"}],"volumes":[{"hostPath":"foo","containerPath":"bar"}],"volumes_from":[{"name":"myvolume"}],"deployment":{"count":1}}`
	json.Unmarshal([]byte(str), &service)

	assert.Equal(t, "myServiceName", service.Name)
	assert.Equal(t, 8080, int(service.Ports[0].ContainerPort))
	assert.Equal(t, "bar", service.Volumes[0].ContainerPath)
	assert.Equal(t, 1, service.Deployment.Count)
	assert.Equal(t, "myvolume", service.VolumesFrom[0].Name)
}
