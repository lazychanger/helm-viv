package engine

import (
	"github.com/lazychanger/helm-variable-in-values/pkg/utils"
	"sigs.k8s.io/yaml"
)

type tree struct {
	data   map[string]interface{}
	top    *tree
	parent *tree
}

func (t *tree) UnmarshalWithYAML(data []byte) error {
	if data == nil {
		t.data = map[string]interface{}{}
		return nil
	}

	return yaml.Unmarshal(data, &t.data)
}

func (t *tree) MarshalWithYAML() ([]byte, error) {
	return yaml.Marshal(&t.data)
}

func (t *tree) Top() *tree {
	return utils.IF(t.top == nil, t, t.top)
}

func (t *tree) Parent() *tree {
	return t.parent
}

func (t *tree) CreateChildAndSelect(node string) *tree {
	subTree, _ := newTree(nil)
	t.data[node] = subTree.data
	subTree.parent = t
	subTree.top = t.Top()

	return subTree
}

func newTree(data []byte) (*tree, error) {
	t := new(tree)
	if err := t.UnmarshalWithYAML(data); err != nil {
		return nil, err
	}
	return t, nil
}
