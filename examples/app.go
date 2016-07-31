package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"time"

	"github.com/gofury/fastjsonapi"
	"github.com/valyala/fasthttp"
)

func createBlog(ctx *fasthttp.RequestCtx) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.create")

	blog := new(Blog)

	if err := jsonapiRuntime.UnmarshalPayload(ctx.PostBody(), blog); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
		return
	}

	// ...do stuff with your blog...

	ctx.SetStatusCode(fasthttp.StatusCreated)
	ctx.SetContentType(fastjsonapi.ContentType)

	if err := jsonapiRuntime.MarshalOnePayload(ctx, blog); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

func listBlogs(ctx *fasthttp.RequestCtx) {
	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.list")
	// ...fetch your blogs, filter, offset, limit, etc...

	// but, for now
	blogs := testBlogsForList()

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetContentType(fastjsonapi.ContentType)
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

	jsonapiRuntime := jsonapi.NewRuntime().Instrument("blogs.show")

	// but, for now
	blog := testBlogForCreate(intID)
	ctx.SetStatusCode(fasthttp.StatusOK)

	ctx.SetConnectionClose(fastjsonapi.ContentType)
	if err := jsonapiRuntime.MarshalOnePayload(w, blog); err != nil {
		ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
	}
}

func main() {
	jsonapi.Instrumentation = func(r *jsonapi.Runtime, eventType jsonapi.Event, callGUID string, dur time.Duration) {
		metricPrefix := r.Value("instrument").(string)

		if eventType == jsonapi.UnmarshalStart {
			fmt.Printf("%s: id, %s, started at %v\n", metricPrefix+".jsonapi_unmarshal_time", callGUID, time.Now())
		}

		if eventType == jsonapi.UnmarshalStop {
			fmt.Printf("%s: id, %s, stopped at, %v , and took %v to unmarshal payload\n", metricPrefix+".jsonapi_unmarshal_time", callGUID, time.Now(), dur)
		}

		if eventType == jsonapi.MarshalStart {
			fmt.Printf("%s: id, %s, started at %v\n", metricPrefix+".jsonapi_marshal_time", callGUID, time.Now())
		}

		if eventType == jsonapi.MarshalStop {
			fmt.Printf("%s: id, %s, stopped at, %v , and took %v to marshal payload\n", metricPrefix+".jsonapi_marshal_time", callGUID, time.Now(), dur)
		}
	}

	http.HandleFunc("/blogs", func(ctx *fasthttp.RequestCtx) {
		if !regexp.MustCompile(`application/vnd\.api\+json`).Match([]byte(r.Header.Get("Accept"))) {
			http.Error(w, "Unsupported Media Type", fasthttp.StatusUnsupportedMediaType)
			return
		}

		if ctx.Method() == byte[]("POST") {
			createBlog(w, r)
		} else if r.FormValue("id") != "" {
			showBlog(w, r)
		} else {
			listBlogs(w, r)
		}
	})

	exerciseHandler()
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

func exerciseHandler() {
	// list
	req, _ := http.NewRequest("GET", "/blogs", nil)

	req.Header.Set("Accept", "application/vnd.api+json")

	w := httptest.NewRecorder()

	fmt.Println("============ start list ===========\n")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop list ===========\n")

	jsonReply, _ := ioutil.ReadAll(w.Body)

	fmt.Println("============ jsonapi response from list ===========\n")
	fmt.Println(string(jsonReply))
	fmt.Println("============== end raw jsonapi from list =============")

	// show
	req, _ = http.NewRequest("GET", "/blogs?id=1", nil)

	req.Header.Set("Accept", "application/vnd.api+json")

	w = httptest.NewRecorder()

	fmt.Println("============ start show ===========\n")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop show ===========\n")

	jsonReply, _ = ioutil.ReadAll(w.Body)

	fmt.Println("\n============ jsonapi response from show ===========\n")
	fmt.Println(string(jsonReply))
	fmt.Println("============== end raw jsonapi from show =============")

	// create
	blog := testBlogForCreate(1)
	in := bytes.NewBuffer(nil)
	jsonapi.MarshalOnePayloadEmbedded(in, blog)

	req, _ = http.NewRequest("POST", "/blogs", in)

	req.Header.Set("Accept", "application/vnd.api+json")

	w = httptest.NewRecorder()

	fmt.Println("============ start create ===========\n")
	http.DefaultServeMux.ServeHTTP(w, req)
	fmt.Println("============ stop create ===========\n")

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, w.Body)

	fmt.Println("\n============ jsonapi response from create ===========\n")
	fmt.Println(buf.String())
	fmt.Println("============== end raw jsonapi response =============")

	responseBlog := new(Blog)

	jsonapi.UnmarshalPayload(buf, responseBlog)

	out := bytes.NewBuffer(nil)
	json.NewEncoder(out).Encode(responseBlog)

	fmt.Println("\n================ Viola! Converted back our Blog struct =================\n")
	fmt.Printf("%s\n", out.Bytes())
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
