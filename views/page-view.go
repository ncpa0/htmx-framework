package views

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/antchfx/xmlquery"
	echo "github.com/labstack/echo/v4"
	"github.com/ncpa0/hardwire/configuration"
	resources "github.com/ncpa0/hardwire/resource-provider"
	"github.com/ncpa0/hardwire/utils"
	. "github.com/ncpa0cpl/convenient-structures"
)

type PageView struct {
	root              string
	title             string
	filepath          string
	routePathname     string
	isDynamic         bool
	requiredResources *Map[string, string]
	queryCache        *Map[string, *NodeProxy]
	queryAllCache     *Map[string, *Array[*NodeProxy]]
	document          *NodeProxy
	Metadata          *pageMetafile
}

type NodeProxy struct {
	parentRoot *PageView
	node       *xmlquery.Node
	raw        string
	etag       string

	// Only present if parent's isDynamic is true
	template *template.Template
}

const PREFIX_LEN = len("<?xml version=\"1.0\"?>")
const TEMPL_QUOTE = "@#34T;"

func nodeToString(node *xmlquery.Node) string {
	s := node.OutputXMLWithOptions(
		xmlquery.WithOutputSelf(),
		xmlquery.WithPreserveSpace(),
	)
	if strings.HasPrefix(s, "<?xml") {
		s = s[PREFIX_LEN:]
	}
	s = strings.ReplaceAll(s, TEMPL_QUOTE, "\"")
	return s
}

func addClass(node *xmlquery.Node, class string) {
	var currentClassAttribute *xmlquery.Attr
	for i, attr := range node.Attr {
		if attr.Name.Local == "class" {
			currentClassAttribute = &node.Attr[i]
			break
		}
	}

	if currentClassAttribute == nil {
		node.Attr = append(node.Attr, xmlquery.Attr{
			Name: xml.Name{
				Local: "class",
			},
			Value: class,
		})
	} else {
		currentClassAttribute.Value += " " + class
	}
}

func NewPageView(root string, filepath string) (*PageView, error) {
	file, err := os.Open(path.Join(root, filepath))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	doc, err := xmlquery.Parse(file)

	if err != nil {
		return nil, err
	}

	dirname := path.Dir(filepath)
	basename := path.Base(strings.TrimSuffix(filepath, ".html"))
	metaFilepath := path.Join(root, dirname, basename+".meta.json")

	metaFile, err := loadPageMetafile(metaFilepath)
	if err != nil {
		return nil, err
	}

	var rawHtml string
	var title string
	var routePathname string = filepath
	rawHtml = nodeToString(doc)

	titleNode := xmlquery.FindOne(doc, "//title")
	if titleNode != nil && titleNode.FirstChild != nil {
		title = titleNode.FirstChild.Data
	}

	if !path.IsAbs(routePathname) {
		routePathname = "/" + routePathname
	}
	if !configuration.Current.KeepExtension {
		routePathname = routePathname[:len(routePathname)-len(path.Ext(routePathname))]
	}

	hash := utils.Hash(rawHtml)

	var templ *template.Template
	if metaFile.IsDynamic {
		templ, err = template.New(filepath).Parse(rawHtml)

		if err != nil {
			return nil, err
		}
	}

	requiredResources := NewMap(map[string]string{})
	for _, res := range metaFile.Resources {
		requiredResources.Set(res.Key, res.Res)
	}

	view := &PageView{
		root:              root,
		title:             title,
		filepath:          filepath,
		routePathname:     routePathname,
		isDynamic:         metaFile.IsDynamic,
		requiredResources: requiredResources,
		queryCache:        NewMap(map[string]*NodeProxy{}),
		queryAllCache:     NewMap(map[string]*Array[*NodeProxy]{}),
		document: &NodeProxy{
			node:     doc,
			raw:      rawHtml,
			etag:     hash,
			template: templ,
		},
		Metadata: metaFile,
	}

	view.document.parentRoot = view

	return view, nil
}

func (v *PageView) GetResourceKeys() *Array[string] {
	return v.requiredResources.Values()
}

func (v *PageView) IsDynamic() bool {
	return v.isDynamic
}

func (v *PageView) GetFilepath() string {
	return v.filepath
}

func (v *PageView) GetRoutePathname() string {
	return v.routePathname
}

func pathSegments(p string) []string {
	segments := strings.Split(p, "/")
	n := 0
	for _, segment := range segments {
		if segment != "" {
			segments[n] = segment
			n++
		}
	}

	return segments[:n]
}

func (v *PageView) MatchesRoute(routePathname string) bool {
	fpathSegments := pathSegments(routePathname)
	viewPathSegments := pathSegments(v.routePathname)

	if len(fpathSegments) == len(viewPathSegments) {
		for i, seg := range fpathSegments {
			viewSeg := viewPathSegments[i]
			if seg != viewSeg && viewSeg[0] != ':' {
				return false
			}
		}
		return true
	}

	return false
}

func (v *PageView) QuerySelector(selector string) *utils.Option[NodeProxy] {
	// first check cache
	if cached, ok := v.queryCache.Get(selector); ok {
		return utils.NewOption(cached)
	}

	query := utils.NewTranslator(selector).XPathQuery()
	result := xmlquery.FindOne(v.document.node, query)

	if result == nil {
		return utils.Empty[NodeProxy]()
	}

	rawHtml := nodeToString(result)

	var templ *template.Template
	if v.isDynamic {
		templ, _ = template.New(v.filepath + query).Parse(rawHtml)
	}

	node := &NodeProxy{
		node:       result,
		raw:        rawHtml,
		etag:       utils.Hash(rawHtml),
		parentRoot: v,
		template:   templ,
	}

	v.queryCache.Set(selector, node)

	return utils.NewOption(node)
}

func (v *PageView) QuerySelectorAll(selector string) []*NodeProxy {
	// first check cache
	if cached, ok := v.queryAllCache.Get(selector); ok {
		return cached.ToSlice()
	}

	query := utils.NewTranslator(selector).XPathQuery()
	nodeList := xmlquery.Find(v.document.node, query)

	result := make([]*NodeProxy, len(nodeList))
	for _, node := range nodeList {
		rawHtml := nodeToString(node)

		var templ *template.Template
		if v.isDynamic {
			templ, _ = template.New(v.filepath + query).Parse(rawHtml)
		}

		node := &NodeProxy{
			node:       node,
			raw:        rawHtml,
			etag:       utils.Hash(rawHtml),
			parentRoot: v,
			template:   templ,
		}
		result = append(result, node)
	}

	v.queryAllCache.Set(selector, NewArray(result).Copy())

	return result
}

type RenderedView struct {
	Html string
	Etag string
}

func (v *PageView) Render(c echo.Context) (*RenderedView, error) {
	return v.document.Render(c)
}

func paramMap(c echo.Context) map[string]string {
	params := make(map[string]string)

	for _, param := range c.ParamNames() {
		params[param] = c.Param(param)
	}

	return params
}

func (node *NodeProxy) Render(c echo.Context) (*RenderedView, error) {
	var rawHtml string
	var etag string

	if node.parentRoot.isDynamic {
		templateData := NewMap(map[string]interface{}{})

		resIterator := node.parentRoot.requiredResources.Iterator()
		for !resIterator.Done() {
			entry, _ := resIterator.Next()

			handler := resources.Provider.Find(entry.Value)
			if handler.IsNil() {
				c.String(404, "resource not found")
				return nil, fmt.Errorf("resource not found: '%s'", entry.Value)
			}

			requestContext := resources.NewDynamicRequestContext(
				c,
				paramMap(c),
				node.parentRoot.routePathname,
			)

			resourceValue, err := handler.Get().Handle(requestContext)
			if err != nil {
				return nil, err
			}

			templateData.Set(entry.Key, resourceValue)
		}

		var buff bytes.Buffer
		err := node.template.Execute(&buff, templateData.ToMap())
		if err != nil {
			return nil, err
		}

		rawHtml = buff.String()

		if configuration.Current.Caching.DynamicRoutes.NoStore {
			etag = ""
		} else {
			etag = utils.Hash(rawHtml)
		}
	} else {
		rawHtml = node.raw
		etag = node.etag
	}

	if node.parentRoot.title != "" {
		result := RenderedView{
			Html: fmt.Sprintf("<title>%s</title>\n%s", node.parentRoot.title, rawHtml),
			Etag: etag,
		}
		return &result, nil
	}

	result := RenderedView{
		Html: rawHtml,
		Etag: etag,
	}

	return &result, nil
}
