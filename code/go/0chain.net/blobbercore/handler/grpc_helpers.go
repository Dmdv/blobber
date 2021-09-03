package handler

import (
	"context"
	"fmt"
	"github.com/0chain/blobber/code/go/0chain.net/core/logging"
	"net/http"

	"github.com/0chain/blobber/code/go/0chain.net/blobbercore/blobbergrpc/proto"
	"github.com/0chain/blobber/code/go/0chain.net/core/common"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type gRPCHeaderMetadata struct {
	Client          string
	ClientKey       string
	ClientSignature string
}

func registerGRPCServices(r *mux.Router, server *grpc.Server) {
	blobberService := newGRPCBlobberService()
	r.Use(Middleware2("ds"))
	grpcGatewayHandler := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(CustomMatcher),
	)

	blobbergrpc.RegisterBlobberServiceServer(server, blobberService)
	err := blobbergrpc.RegisterBlobberServiceHandlerServer(context.Background(), grpcGatewayHandler, blobberService)
	if err != nil {
		logging.Logger.Error("Error registering blobber service handler" + err.Error())
		return
	}

	r.PathPrefix("/").Handler(grpcGatewayHandler)

	err = grpcGatewayHandler.HandlePath("POST", "/v1/file/upload/{allocation}",
		func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			r = mux.SetURLVars(r, map[string]string{"allocation": pathParams[`allocation`]})
			common.UserRateLimit(common.ToJSONResponse(WithConnection(UploadHandler)))(w, r)
		})
	if err != nil {
		logging.Logger.Error("Error registering upload POST handler" + err.Error())
		return
	}

	err = grpcGatewayHandler.HandlePath("PUT", "/v1/file/upload/{allocation}",
		func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			r = mux.SetURLVars(r, map[string]string{"allocation": pathParams[`allocation`]})
			common.UserRateLimit(common.ToJSONResponse(WithConnection(UploadHandler)))(w, r)
		})
	if err != nil {
		logging.Logger.Error("Error registering upload PUT handler" + err.Error())
		return
	}

	err = grpcGatewayHandler.HandlePath("DELETE", "/v1/file/upload/{allocation}",
		func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			r = mux.SetURLVars(r, map[string]string{"allocation": pathParams[`allocation`]})
			common.UserRateLimit(common.ToJSONResponse(WithConnection(UploadHandler)))(w, r)
		})
	if err != nil {
		logging.Logger.Error("Error registering upload DELETE handler" + err.Error())
		return
	}

	err = grpcGatewayHandler.HandlePath("POST", "/v1/file/download/{allocation}",
		func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
			r = mux.SetURLVars(r, map[string]string{"allocation": pathParams[`allocation`]})
			common.UserRateLimit(common.ToByteStream(WithConnection(DownloadHandler)))(w, r)
		})

	if err != nil {
		logging.Logger.Error("Error registering download POST handler" + err.Error())
		return
	}
}

func Middleware2(s string) mux.MiddlewareFunc {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// do stuff
			fmt.Println(s)
			h.ServeHTTP(w, r)
		})
	}
}

func getGRPCMetaDataFromCtx(ctx context.Context) *gRPCHeaderMetadata {
	metaData := &gRPCHeaderMetadata{}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return metaData
	}

	getMetaData := func(key string) string {
		list := md.Get(key)
		if len(list) > 0 {
			return list[0]
		}
		return ""
	}

	metaData.Client = getMetaData(common.ClientHeader)
	metaData.ClientKey = getMetaData(common.ClientKeyHeader)
	metaData.ClientSignature = getMetaData(common.ClientSignatureHeader)
	return metaData
}

func httpRequestWithMetaData(r *http.Request, md *gRPCHeaderMetadata, alloc string) {
	r.Header.Set(common.ClientHeader, md.Client)
	r.Header.Set(common.ClientKeyHeader, md.ClientKey)
	r.Header.Set(common.ClientSignatureHeader, md.ClientSignature)
	*r = *mux.SetURLVars(r, map[string]string{"allocation": alloc})
}

func CustomMatcher(key string) (string, bool) {
	switch key {
	case common.ClientHeader:
		return key, true
	case common.ClientKeyHeader:
		return key, true
	case common.ClientSignatureHeader:
		return key, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}