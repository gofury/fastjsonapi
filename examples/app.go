package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
	"encoding/json"

	"github.com/gofury/fastjsonapi"
	"github.com/valyala/fasthttp"
	"bytes"
)

func createBlog(ctx *fasthttp.RequestCtx) {
	jsonapiRuntime := fastjsonapi.NewRuntime().Instrument("blogs.create")

	blog := new(Blog)

	if err := jsonapiRuntime.UnmarshalPayload(ctx.PostBody(), blog); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	// ...do stuff with your blog...

	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetContentTypeBytes(fastjsonapi.ContentType)

	if err := jsonapiRuntime.MarshalOnePayload(ctx, blog); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

func listBlogs(ctx *fasthttp.RequestCtx) {
	jsonapiRuntime := fastjsonapi.NewRuntime().Instrument("blogs.list")
	// ...fetch your blogs, filter, offset, limit, etc...

	// but, for now
	blogs := testBlogsForList()

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes(fastjsonapi.ContentType)
	if err := jsonapiRuntime.MarshalManyPayload(ctx, blogs); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

func showBlog(ctx *fasthttp.RequestCtx) {
	id := string(ctx.FormValue("id"))

	// ...fetch your blog...

	intID, err := strconv.Atoi(id)
	if err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	jsonapiRuntime := fastjsonapi.NewRuntime().Instrument("blogs.show")

	// but, for now
	blog := testBlogForCreate(intID)
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentTypeBytes(fastjsonapi.ContentType)

	if err := jsonapiRuntime.MarshalOnePayload(ctx, blog); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

func main() {
	fastjsonapi.Instrumentation = func(r *fastjsonapi.Runtime, eventType fastjsonapi.Event, callGUID string, dur time.Duration) {
		metricPrefix := r.Value("instrument").(string)

		if eventType == fastjsonapi.UnmarshalStart {
			fmt.Printf("%s: id, %s, started at %v\n", metricPrefix+".jsonapi_unmarshal_time", callGUID, time.Now())
		}

		if eventType == fastjsonapi.UnmarshalStop {
			fmt.Printf("%s: id, %s, stopped at, %v , and took %v to unmarshal payload\n", metricPrefix+".jsonapi_unmarshal_time", callGUID, time.Now(), dur)
		}

		if eventType == fastjsonapi.MarshalStart {
			fmt.Printf("%s: id, %s, started at %v\n", metricPrefix+".jsonapi_marshal_time", callGUID, time.Now())
		}

		if eventType == fastjsonapi.MarshalStop {
			fmt.Printf("%s: id, %s, stopped at, %v , and took %v to marshal payload\n", metricPrefix+".jsonapi_marshal_time", callGUID, time.Now(), dur)
		}
	}

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		if ! regexp.MustCompile(`application/vnd\.api\+json`).Match(ctx.Request.Header.Peek("Accept")) {
			ctx.Error("Unsupported Media Type", fasthttp.StatusUnsupportedMediaType)
			return
		} else if ( !bytes.Contains(ctx.RequestURI(), []byte("/blogs")) ) {
			ctx.Error("Resource not found", fasthttp.StatusNotFound)
		}

		if bytes.Equal(ctx.Method(), []byte("POST")) {
			createBlog(ctx)
		} else if ctx.FormValue("id") != nil {
			showBlog(ctx)
		} else {
			listBlogs(ctx)
		}
	}

	exerciseHandler(requestHandler)
}

func testBlogForCreate(i int) *Blog {
	return &Blog{
		ID:        1 * i,
		Title:     "Title 1",
		CreatedAt: time.Now(),
		Posts: []*Post{
			&Post{
				ID:    1 * i,
				Title: "Foo",
				Body:  "Bar",
				Comments: []*Comment{
					&Comment{
						ID:   1 * i,
						Body: "foo",
					},
					&Comment{
						ID:   2 * i,
						Body: "bar",
					},
				},
			},
			&Post{
				ID:    2 * i,
				Title: "Fuubar",
				Body:  "Bas",
				Comments: []*Comment{
					&Comment{
						ID:   1 * i,
						Body: "foo",
					},
					&Comment{
						ID:   3 * i,
						Body: "bas",
					},
				},
			},
		},
		CurrentPost: &Post{
			ID:    1 * i,
			Title: "Foo",
			Body:  "Bar",
			Comments: []*Comment{
				&Comment{
					ID:   1 * i,
					Body: "foo",
				},
				&Comment{
					ID:   2 * i,
					Body: "bar",
				},
			},
		},
	}
}

func testBlogsForList() []interface{} {
	blogs := make([]interface{}, 0, 10)

	for i := 0; i < 10; i += 1 {
		blogs = append(blogs, testBlogForCreate(i))
	}

	return blogs
}

func exerciseHandler(handler fasthttp.RequestHandler) {
	// list
	ctx := newRequest("GET", "/blogs", nil)

	fmt.Println("============ start list ===========\n")
	handler(ctx)
	fmt.Println("============ stop list ===========\n")

	fmt.Println("============ jsonapi response from list ===========\n")
	fmt.Println(string(ctx.Response.Body()))
	fmt.Println("============== end raw fastjsonapi from list =============")

	// show
	ctx = newRequest("GET", "/blogs?id=1", nil)

	fmt.Println("============ start show ===========\n")
	handler(ctx)
	fmt.Println("============ stop show ===========\n")

	fmt.Println("\n============ jsonapi response from show ===========\n")
	fmt.Println(string(ctx.Response.Body()))
	fmt.Println("============== end raw fastjsonapi from show =============")

	// create
	blog := testBlogForCreate(1)
	in := bytes.NewBuffer(nil)
	fastjsonapi.MarshalOnePayloadEmbedded(in, blog)

	ctx = newRequest("POST", "/blogs", in.Bytes())

	fmt.Println("============ start create ===========\n")
	handler(ctx)
	fmt.Println("============ stop create ===========\n")

	fmt.Println("\n============ jsonapi response from create ===========\n")
	fmt.Println(string(ctx.Response.Body()))
	fmt.Println("============== end raw jsonapi response =============")

	responseBlog := new(Blog)

	fastjsonapi.UnmarshalPayload(ctx.Response.Body(), responseBlog)

	blogJson, _ := json.Marshal(responseBlog)

	fmt.Println("\n================ Viola! Converted back our Blog struct =================\n")
	fmt.Printf("%s\n", blogJson)
	fmt.Println("================ end marshal materialized Blog struct =================")
}

type Blog struct {
	ID            int       `jsonapi:"primary,blogs"`
	Title         string    `jsonapi:"attr,title"`
	Posts         []*Post   `jsonapi:"relation,posts"`
	CurrentPost   *Post     `jsonapi:"relation,current_post"`
	CurrentPostID int       `jsonapi:"attr,current_post_id"`
	CreatedAt     time.Time `jsonapi:"attr,created_at"`
	ViewCount     int       `jsonapi:"attr,view_count"`
}

type Post struct {
	ID       int        `jsonapi:"primary,posts"`
	BlogID   int        `jsonapi:"attr,blog_id"`
	Title    string     `jsonapi:"attr,title"`
	Body     string     `jsonapi:"attr,body"`
	Comments []*Comment `jsonapi:"relation,comments"`
}

type Comment struct {
	ID     int    `jsonapi:"primary,comments"`
	PostID int    `jsonapi:"attr,post_id"`
	Body   string `jsonapi:"attr,body"`
}

func newRequest(method string, uri string, data []byte) *fasthttp.RequestCtx {
	ctx := fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	ctx.Request.SetBody(data)
	ctx.Request.Header.SetBytesV("Accept", fastjsonapi.ContentType)
	return &ctx
}