package views

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/ncpa0/htmx-framework/utils"
	"golang.org/x/net/html"
)

type View struct {
	root          string
	filepath      string
	document      *NodeProxy
	queryCache    map[string]*NodeProxy
	queryAllCache map[string][]*NodeProxy
}

type NodeProxy struct {
	node *html.Node
	raw  string
	etag string
}

func NewView(root string, filepath string) (*View, error) {
	doc, err := htmlquery.LoadDoc(path.Join(root, filepath))

	if err != nil {
		fmt.Println("Error loading view.")
		fmt.Println(err)
		return nil, err
	}

	var b bytes.Buffer
	err = html.Render(&b, doc)

	if err != nil {
		return nil, err
	}

	rawHtml := b.String()
	hash := utils.Hash(rawHtml)

	return &View{
		root:     root,
		filepath: filepath,
		document: &NodeProxy{
			node: doc,
			raw:  b.String(),
			etag: hash,
		},
		queryCache:    make(map[string]*NodeProxy),
		queryAllCache: make(map[string][]*NodeProxy),
	}, nil
}

func (v *View) GetFilepath() string {
	return v.filepath
}

func (v *View) FilepathMatches(fpath string) bool {
	if v.filepath == fpath {
		return true
	}

	fullPath := path.Join(v.root, v.filepath)
	return fullPath == fpath || strings.HasSuffix(fullPath, fpath)
}

func (v *View) QuerySelector(selector string) *utils.Option[NodeProxy] {
	// first check cache
	if cached, ok := v.queryCache[selector]; ok {
		return utils.NewOption(cached)
	}

	query := utils.NewTranslator(selector).XPathQuery()
	result := htmlquery.FindOne(v.document.node, query)

	if result == nil {
		return utils.Empty[NodeProxy]()
	}

	var b bytes.Buffer
	html.Render(&b, result)
	rawHtml := b.String()
	node := &NodeProxy{
		node: result,
		raw:  rawHtml,
		etag: utils.Hash(rawHtml),
	}

	return utils.NewOption(node)
}

func (v *View) QuerySelectorAll(selector string) []*NodeProxy {
	// first check cache
	if cached, ok := v.queryAllCache[selector]; ok {
		result := make([]*NodeProxy, len(cached))
		copy(result, cached)
		return result
	}

	query := utils.NewTranslator(selector).XPathQuery()
	nodeList := htmlquery.Find(v.document.node, query)

	result := make([]*NodeProxy, len(nodeList))
	for _, node := range nodeList {
		var b bytes.Buffer
		html.Render(&b, node)
		rawHtml := b.String()
		node := &NodeProxy{
			node: node,
			raw:  rawHtml,
			etag: utils.Hash(rawHtml),
		}
		result = append(result, node)
	}

	return result
}

func (v *View) GetNode() *NodeProxy {
	return v.document
}

func (n *NodeProxy) ToHtml() string {
	return n.raw
}

func (n *NodeProxy) GetEtag() string {
	return n.etag
}
