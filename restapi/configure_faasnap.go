// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/ucsdsysnet/faasnap/daemon"
	"github.com/ucsdsysnet/faasnap/models"
	"github.com/ucsdsysnet/faasnap/restapi/operations"

	"contrib.go.opencensus.io/exporter/zipkin"
	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

//go:generate swagger generate server --target ../../faasnap --name faasnap --spec ../swagger.json --principal interface{}

func configureFlags(api *operations.FaasnapAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.FaasnapAPI) http.Handler {
	// configure the api here
	log.Printf("configuring api")
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	api.DeleteVmsVMIDHandler = operations.DeleteVmsVMIDHandlerFunc(func(params operations.DeleteVmsVMIDParams) middleware.Responder {
		if err := daemon.StopVM(params.HTTPRequest, params.VMID); err != nil {
			return operations.NewDeleteVmsVMIDBadRequest().WithPayload(&operations.DeleteVmsVMIDBadRequestBody{Message: err.Error()})
		}
		return operations.NewDeleteVmsVMIDOK()
	})

	if api.GetFunctionsHandler == nil {
		api.GetFunctionsHandler = operations.GetFunctionsHandlerFunc(func(params operations.GetFunctionsParams) middleware.Responder {
			return middleware.NotImplemented("operation operations.GetFunctions has not yet been implemented")
		})
	}
	if api.GetVmsHandler == nil {
		api.GetVmsHandler = operations.GetVmsHandlerFunc(func(params operations.GetVmsParams) middleware.Responder {
			return middleware.NotImplemented("operation operations.GetVms has not yet been implemented")
		})
	}
	if api.GetVmsVMIDHandler == nil {
		api.GetVmsVMIDHandler = operations.GetVmsVMIDHandlerFunc(func(params operations.GetVmsVMIDParams) middleware.Responder {
			return middleware.NotImplemented("operation operations.GetVmsVMID has not yet been implemented")
		})
	}
	api.PostFunctionsHandler = operations.PostFunctionsHandlerFunc(func(params operations.PostFunctionsParams) middleware.Responder {
		if err := daemon.CreateFunction(params); err != nil {
			return operations.NewPostFunctionsBadRequest().WithPayload(&operations.PostFunctionsBadRequestBody{Message: err.Error()})
		}
		return operations.NewPostFunctionsOK()
	})
	api.PostInvocationsHandler = operations.PostInvocationsHandlerFunc(func(params operations.PostInvocationsParams) middleware.Responder {
		// intLoadMincore := make([]int, len(params.Invocation.LoadMincore))
		// for i, v := range params.Invocation.LoadMincore {
		// 	intLoadMincore[i] = int(v)
		// }
		result, vmId, traceId, err := daemon.InvokeFunction(params.HTTPRequest, params.Invocation)
		if err != nil {
			return operations.NewPostInvocationsBadRequest().WithPayload(&operations.PostInvocationsBadRequestBody{Message: err.Error()})
		}
		return operations.NewPostInvocationsOK().WithPayload(&operations.PostInvocationsOKBody{
			Duration: 0, VMID: vmId, Result: result, TraceID: traceId})
	})
	api.PostSnapshotsHandler = operations.PostSnapshotsHandlerFunc(func(params operations.PostSnapshotsParams) middleware.Responder {
		ssId, err := daemon.TakeSnapshot(params.HTTPRequest, *params.Snapshot.VMID, params.Snapshot.SnapshotType, params.Snapshot.SnapshotPath,
			params.Snapshot.MemFilePath, params.Snapshot.Version, params.Snapshot.RecordRegions, int(params.Snapshot.SizeThreshold), int(params.Snapshot.IntervalThreshold))
		if err != nil {
			return &operations.PostSnapshotsBadRequest{Payload: &operations.PostSnapshotsBadRequestBody{Message: err.Error()}}
		}
		ret := *params.Snapshot
		ret.SsID = ssId
		return &operations.PostSnapshotsOK{Payload: &ret}
	})

	api.PutSnapshotsHandler = operations.PutSnapshotsHandlerFunc(func(params operations.PutSnapshotsParams) middleware.Responder {
		snap, err := daemon.CopySnapshot(params.HTTPRequest.Context(), params.FromSnapshot, params.MemFilePath)
		if err != nil {
			return &operations.PutSnapshotsBadRequest{Payload: &operations.PutSnapshotsBadRequestBody{Message: err.Error()}}
		}
		return &operations.PutSnapshotsOK{Payload: snap}
	})
	api.PatchSnapshotsSsIDHandler = operations.PatchSnapshotsSsIDHandlerFunc(func(params operations.PatchSnapshotsSsIDParams) middleware.Responder {
		if err := daemon.ChangeSnapshot(params.HTTPRequest, params.SsID, params.State.DigHole, params.State.LoadCache, params.State.DropCache); err != nil {
			return &operations.PatchSnapshotsSsIDBadRequest{Payload: &operations.PatchSnapshotsSsIDBadRequestBody{Message: err.Error()}}
		}
		return &operations.PatchSnapshotsSsIDOK{}
	})

	api.GetSnapshotsSsIDMincoreHandler = operations.GetSnapshotsSsIDMincoreHandlerFunc(func(params operations.GetSnapshotsSsIDMincoreParams) middleware.Responder {
		state, err := daemon.GetMincore(params.HTTPRequest, params.SsID)
		if err != nil {
			return &operations.GetSnapshotsSsIDMincoreBadRequest{Payload: &operations.GetSnapshotsSsIDMincoreBadRequestBody{Message: err.Error()}}
		}
		return &operations.GetSnapshotsSsIDMincoreOK{Payload: state}
	})
	api.PutSnapshotsSsIDMincoreHandler = operations.PutSnapshotsSsIDMincoreHandlerFunc(func(params operations.PutSnapshotsSsIDMincoreParams) middleware.Responder {
		if err := daemon.CopyMincore(params.HTTPRequest, params.SsID, *params.Source); err != nil {
			return &operations.PutSnapshotsSsIDMincoreBadRequest{Payload: &operations.PutSnapshotsSsIDMincoreBadRequestBody{Message: err.Error()}}
		}
		return &operations.PutSnapshotsSsIDMincoreOK{}
	})
	api.PostSnapshotsSsIDMincoreHandler = operations.PostSnapshotsSsIDMincoreHandlerFunc(func(params operations.PostSnapshotsSsIDMincoreParams) middleware.Responder {
		if err := daemon.AddMincoreLayer(params.HTTPRequest, params.SsID, int(params.Layer.Position), params.Layer.FromDiff); err != nil {
			return &operations.PostSnapshotsSsIDMincoreBadRequest{Payload: &operations.PostSnapshotsSsIDMincoreBadRequestBody{Message: err.Error()}}
		}
		return &operations.PostSnapshotsSsIDMincoreOK{}
	})
	api.PatchSnapshotsSsIDMincoreHandler = operations.PatchSnapshotsSsIDMincoreHandlerFunc(func(params operations.PatchSnapshotsSsIDMincoreParams) middleware.Responder {
		if err := daemon.ChangeMincoreState(params.HTTPRequest.Context(), params.SsID, int(params.State.FromRecordsSize), params.State.TrimRegions, params.State.ToWsFile, params.State.InactiveWs, params.State.ZeroWs, int(params.State.SizeThreshold), int(params.State.IntervalThreshold), params.State.MincoreCache, params.State.DropWsCache); err != nil {
			return &operations.PatchSnapshotsSsIDMincoreBadRequest{Payload: &operations.PatchSnapshotsSsIDMincoreBadRequestBody{Message: err.Error()}}
		}
		return &operations.PatchSnapshotsSsIDMincoreOK{}
	})
	api.PostVmsHandler = operations.PostVmsHandlerFunc(func(params operations.PostVmsParams) middleware.Responder {
		vmId, err := daemon.StartVM(params.HTTPRequest, params.VM.FuncName, params.VM.SsID, params.VM.Namespace)
		if err != nil {
			return operations.NewPostVmsBadRequest().WithPayload(&operations.PostVmsBadRequestBody{Message: err.Error()})
		}
		return operations.NewPostVmsOK().WithPayload(&models.VM{VMID: &vmId})
	})

	api.PostVmmsHandler = operations.PostVmmsHandlerFunc(func(params operations.PostVmmsParams) middleware.Responder {
		vmId, err := daemon.StartVMM(params.HTTPRequest.Context(), params.VMM.EnableReap, params.VMM.Namespace)
		if err != nil {
			return operations.NewPostVmmsBadRequest().WithPayload(&operations.PostVmmsBadRequestBody{Message: err.Error()})
		}
		return operations.NewPostVmmsOK().WithPayload(&models.VM{VMID: &vmId})
	})

	api.PatchSnapshotsSsIDReapHandler = operations.PatchSnapshotsSsIDReapHandlerFunc(func(params operations.PatchSnapshotsSsIDReapParams) middleware.Responder {
		if err := daemon.ChangeReapCacheState(params.HTTPRequest, params.SsID, params.Cache); err != nil {
			return operations.NewPatchSnapshotsSsIDReapBadRequest().WithPayload(&operations.PatchSnapshotsSsIDReapBadRequestBody{Message: err.Error()})
		}
		return operations.NewPatchSnapshotsSsIDReapOK()
	})

	api.PutNetIfacesNamespaceHandler = operations.PutNetIfacesNamespaceHandlerFunc(func(params operations.PutNetIfacesNamespaceParams) middleware.Responder {
		err := daemon.PutNetwork(params.HTTPRequest, params.Namespace, params.Interface.HostDevName, params.Interface.IfaceID, params.Interface.GuestMac, params.Interface.GuestAddr, params.Interface.UniqueAddr)
		if err != nil {
			return &operations.PutNetIfacesNamespaceBadRequest{Payload: &operations.PutNetIfacesNamespaceBadRequestBody{Message: err.Error()}}
		}
		return operations.NewPutNetIfacesNamespaceOK()
	})

	// api.GetUIHandler = operations.GetUIHandlerFunc(func(params operations.GetUIParams) middleware.Responder {
	// 	return CustomResponder(func(w http.ResponseWriter, _ runtime.Producer) {
	// 		http.FileServer(http.Dir("/ui/build/")).ServeHTTP(w, params.HTTPRequest)
	// 	})
	// })

	api.GetUIDataHandler = operations.GetUIDataHandlerFunc(func(params operations.GetUIDataParams) middleware.Responder {
		return CustomResponder(func(w http.ResponseWriter, _ runtime.Producer) {
			handleUiData(w, params.HTTPRequest)
		})
	})

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
	state = daemon.Setup(s, scheme, addr)
}

func registerZipkin(zipkinHost string, port int) {
	localEndpoint, err := openzipkin.NewEndpoint("daemon", fmt.Sprintf("0.0.0.0:%v", port))
	if err != nil {
		log.Fatalf("Failed to create Zipkin exporter: %v", err)
	}
	reporter := zipkinHTTP.NewReporter(fmt.Sprintf("%v/api/v2/spans", zipkinHost))
	exporter := zipkin.NewExporter(reporter, localEndpoint)
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return FileServerMiddleware(handler)
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	var port = 8080
	var zipkinHost = "http://localhost:9411"

	registerZipkin(zipkinHost, port)
	h := &ochttp.Handler{Handler: handler}
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		log.Fatal("Failed to register ochttp.DefaultServerViews")
	}

	return h
}

type CustomResponder func(http.ResponseWriter, runtime.Producer)

func (c CustomResponder) WriteResponse(w http.ResponseWriter, p runtime.Producer) {
	c(w, p)
}

var state *daemon.DaemonState

func handleUiData(w http.ResponseWriter, req *http.Request) {
	marshalled, err := json.Marshal(state)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("failed to get data, %v", err)))
	}
	w.WriteHeader(200)
	if _, err := w.Write(marshalled); err != nil {
		log.Println("Failed to write data")
	}
}

func FileServerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving", r.Method, r.URL.Path)
		if r.URL.Path == "/ui" {
			log.Println("serving via file server:", r.URL.Path)
			http.FileServer(http.Dir("./ui/build/")).ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
