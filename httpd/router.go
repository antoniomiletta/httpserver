package httpd

type Router struct {
	routes map[string]HandlerFunc
}

type HandlerFunc func(w ResponseWriter, r *Request)

func NewRouter() *Router {
	return &Router{
		routes: make(map[string]HandlerFunc),
	}
}

func (r *Router) resolve(target string) HandlerFunc {
	fn, ok := r.routes[target]
	if !ok {
		return func(w ResponseWriter, r *Request) {
			w.WriteHeader(StatusNotFound)
			w.Write([]byte("Not Found."))
		}
	}

	return fn
}

func (r *Router) Get(target string, fn HandlerFunc) {
	r.routes["GET "+target] = fn
}

func (r *Router) Post(target string, fn HandlerFunc) {
	r.routes["POST "+target] = fn
}

func (r *Router) Put(target string, fn HandlerFunc) {
	r.routes["PUT "+target] = fn
}
func (r *Router) Patch(target string, fn HandlerFunc) {
	r.routes["PATCH "+target] = fn
}

func (r *Router) Delete(target string, fn HandlerFunc) {
	r.routes["DELETE "+target] = fn
}
