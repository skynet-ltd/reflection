package mapping_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/skynet-ltd/reflection/mapping"
)

func TestRefmapping(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Refmapping Suite")
}

var _ = Describe("test diff for HWare model", func() {

	Specify("test Reflection func", func() {
		type Node struct {
			Name     string `tag:"test" diff:"+"`
			Value    uint16
			Siblings []*Node
			Parent   *Node
			Left     *Node
			Right    *Node
			Map      map[uint16]map[string]interface{} `tag:"test" diff:"+"`
			//Map  map[uint16]interface{}
			Data []byte
		}

		testNode := &Node{
			Name:     "First",
			Value:    20,
			Siblings: []*Node{&Node{Name: "Antony"}, &Node{Name: "John"}},
			Parent:   &Node{Name: "Parent", Parent: &Node{Name: "Grand Parent"}},
			Left:     &Node{Name: "Normal", Siblings: []*Node{&Node{Name: "Unknown"}}},
			Right:    nil,
			Map:      map[uint16]map[string]interface{}{12: map[string]interface{}{"World": 20}, 20: map[string]interface{}{"Hello": 10}},
			//Map:  map[uint16]interface{}{12: "World", 10: 10},
			Data: []byte("test data"),
		}

		refs, err := mapping.Reflection(testNode)
		Expect(err).To(BeNil())
		for k, v := range refs {
			fmt.Println(k, v)
		}
	})

})
