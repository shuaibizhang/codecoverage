package coverage

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

func RegisterCoverageServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) (err error) {
	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				// Log error
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				// Log error
			}
		}()
	}()

	return RegisterCoverageServiceHandler(ctx, mux, conn)
}

func RegisterCoverageServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return RegisterCoverageServiceHandlerClient(ctx, mux, NewCoverageServiceClient(conn))
}

func RegisterCoverageServiceHandlerClient(ctx context.Context, mux *runtime.ServeMux, client CoverageServiceClient) error {
	mux.Handle("GET", pattern_CoverageService_GetReportInfo_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		_, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req, "/api.v1.coverage.CoverageService/GetReportInfo", runtime.WithHTTPPathPattern("/api/v1/coverage/report"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, err := client.GetReportInfo(rctx, &GetReportInfoRequest{
			Module: req.URL.Query().Get("module"),
			Branch: req.URL.Query().Get("branch"),
			Commit: req.URL.Query().Get("commit"),
			Type:   req.URL.Query().Get("type"),
		})
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload, _ := json.Marshal(resp)
		w.Write(payload)
	})

	mux.Handle("GET", pattern_CoverageService_GetTreeNodes_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		_, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req, "/api.v1.coverage.CoverageService/GetTreeNodes", runtime.WithHTTPPathPattern("/api/v1/coverage/tree"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, err := client.GetTreeNodes(rctx, &GetTreeNodesRequest{
			ReportId: req.URL.Query().Get("report_id"),
			Path:     req.URL.Query().Get("path"),
		})
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload, _ := json.Marshal(resp)
		w.Write(payload)
	})

	mux.Handle("GET", pattern_CoverageService_GetFileCoverage_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()
		_, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
		rctx, err := runtime.AnnotateContext(ctx, mux, req, "/api.v1.coverage.CoverageService/GetFileCoverage", runtime.WithHTTPPathPattern("/api/v1/coverage/file"))
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, err := client.GetFileCoverage(rctx, &GetFileCoverageRequest{
			ReportId: req.URL.Query().Get("report_id"),
			Path:     req.URL.Query().Get("path"),
		})
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		payload, _ := json.Marshal(resp)
		w.Write(payload)
	})

	return nil
}

var (
	pattern_CoverageService_GetReportInfo_0   = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3}, []string{"api", "v1", "coverage", "report"}, ""))
	pattern_CoverageService_GetTreeNodes_0    = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3}, []string{"api", "v1", "coverage", "tree"}, ""))
	pattern_CoverageService_GetFileCoverage_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2, 2, 3}, []string{"api", "v1", "coverage", "file"}, ""))
)

var (
	forward_CoverageService_GetReportInfo_0   = runtime.ForwardResponseMessage
	forward_CoverageService_GetTreeNodes_0    = runtime.ForwardResponseMessage
	forward_CoverageService_GetFileCoverage_0 = runtime.ForwardResponseMessage
)
