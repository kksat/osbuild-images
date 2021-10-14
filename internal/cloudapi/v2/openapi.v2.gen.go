// Package v2 provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen DO NOT EDIT.
package v2

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"github.com/deepmap/oapi-codegen/pkg/runtime"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

// AWSEC2UploadOptions defines model for AWSEC2UploadOptions.
type AWSEC2UploadOptions struct {
	Region            string   `json:"region"`
	ShareWithAccounts []string `json:"share_with_accounts"`
	SnapshotName      *string  `json:"snapshot_name,omitempty"`
}

// AWSEC2UploadStatus defines model for AWSEC2UploadStatus.
type AWSEC2UploadStatus struct {
	Ami    string `json:"ami"`
	Region string `json:"region"`
}

// AWSS3UploadOptions defines model for AWSS3UploadOptions.
type AWSS3UploadOptions struct {
	Region string `json:"region"`
}

// AWSS3UploadStatus defines model for AWSS3UploadStatus.
type AWSS3UploadStatus struct {
	Url string `json:"url"`
}

// AzureUploadOptions defines model for AzureUploadOptions.
type AzureUploadOptions struct {

	// Name of the uploaded image. It must be unique in the given resource group.
	// If name is omitted from the request, a random one based on a UUID is
	// generated.
	ImageName *string `json:"image_name,omitempty"`

	// Location where the image should be uploaded and registered.
	// How to list all locations:
	// https://docs.microsoft.com/en-us/cli/azure/account?view=azure-cli-latest#az_account_list_locations'
	Location string `json:"location"`

	// Name of the resource group where the image should be uploaded.
	ResourceGroup string `json:"resource_group"`

	// ID of subscription where the image should be uploaded.
	SubscriptionId string `json:"subscription_id"`

	// ID of the tenant where the image should be uploaded.
	// How to find it in the Azure Portal:
	// https://docs.microsoft.com/en-us/azure/active-directory/fundamentals/active-directory-how-to-find-tenant
	TenantId string `json:"tenant_id"`
}

// AzureUploadStatus defines model for AzureUploadStatus.
type AzureUploadStatus struct {
	ImageName string `json:"image_name"`
}

// ComposeId defines model for ComposeId.
type ComposeId struct {
	// Embedded struct due to allOf(#/components/schemas/ObjectReference)
	ObjectReference
	// Embedded fields due to inline allOf schema
	Id string `json:"id"`
}

// ComposeMetadata defines model for ComposeMetadata.
type ComposeMetadata struct {
	// Embedded struct due to allOf(#/components/schemas/ObjectReference)
	ObjectReference
	// Embedded fields due to inline allOf schema

	// ID (hash) of the built commit
	OstreeCommit *string `json:"ostree_commit,omitempty"`

	// Package list including NEVRA
	Packages *[]PackageMetadata `json:"packages,omitempty"`
}

// ComposeRequest defines model for ComposeRequest.
type ComposeRequest struct {
	// Embedded struct due to allOf(#/components/schemas/ObjectReference)
	ObjectReference
	// Embedded fields due to inline allOf schema
	Customizations *Customizations `json:"customizations,omitempty"`
	Distribution   string          `json:"distribution"`
	ImageRequest   ImageRequest    `json:"image_request"`
}

// ComposeStatus defines model for ComposeStatus.
type ComposeStatus struct {
	// Embedded struct due to allOf(#/components/schemas/ObjectReference)
	ObjectReference
	// Embedded fields due to inline allOf schema
	ImageStatus ImageStatus `json:"image_status"`
}

// Customizations defines model for Customizations.
type Customizations struct {
	Packages     *[]string     `json:"packages,omitempty"`
	Subscription *Subscription `json:"subscription,omitempty"`
	Users        *[]User       `json:"users,omitempty"`
}

// Error defines model for Error.
type Error struct {
	// Embedded struct due to allOf(#/components/schemas/ObjectReference)
	ObjectReference
	// Embedded fields due to inline allOf schema
	Code        string `json:"code"`
	OperationId string `json:"operation_id"`
	Reason      string `json:"reason"`
}

// ErrorList defines model for ErrorList.
type ErrorList struct {
	// Embedded struct due to allOf(#/components/schemas/List)
	List
	// Embedded fields due to inline allOf schema
	Items []Error `json:"items"`
}

// GCPUploadOptions defines model for GCPUploadOptions.
type GCPUploadOptions struct {

	// Name of an existing STANDARD Storage class Bucket.
	Bucket string `json:"bucket"`

	// The name to use for the imported and shared Compute Engine image.
	// The image name must be unique within the GCP project, which is used
	// for the OS image upload and import. If not specified a random
	// 'composer-api-<uuid>' string is used as the image name.
	ImageName *string `json:"image_name,omitempty"`

	// The GCP region where the OS image will be imported to and shared from.
	// The value must be a valid GCP location. See https://cloud.google.com/storage/docs/locations.
	// If not specified, the multi-region location closest to the source
	// (source Storage Bucket location) is chosen automatically.
	Region string `json:"region"`

	// List of valid Google accounts to share the imported Compute Engine image with.
	// Each string must contain a specifier of the account type. Valid formats are:
	//   - 'user:{emailid}': An email address that represents a specific
	//     Google account. For example, 'alice@example.com'.
	//   - 'serviceAccount:{emailid}': An email address that represents a
	//     service account. For example, 'my-other-app@appspot.gserviceaccount.com'.
	//   - 'group:{emailid}': An email address that represents a Google group.
	//     For example, 'admins@example.com'.
	//   - 'domain:{domain}': The G Suite domain (primary) that represents all
	//     the users of that domain. For example, 'google.com' or 'example.com'.
	// If not specified, the imported Compute Engine image is not shared with any
	// account.
	ShareWithAccounts *[]string `json:"share_with_accounts,omitempty"`
}

// GCPUploadStatus defines model for GCPUploadStatus.
type GCPUploadStatus struct {
	ImageName string `json:"image_name"`
	ProjectId string `json:"project_id"`
}

// ImageRequest defines model for ImageRequest.
type ImageRequest struct {
	Architecture  string        `json:"architecture"`
	ImageType     ImageTypes    `json:"image_type"`
	Ostree        *OSTree       `json:"ostree,omitempty"`
	Repositories  []Repository  `json:"repositories"`
	UploadOptions UploadOptions `json:"upload_options"`
}

// ImageStatus defines model for ImageStatus.
type ImageStatus struct {
	Status       ImageStatusValue `json:"status"`
	UploadStatus *UploadStatus    `json:"upload_status,omitempty"`
}

// ImageStatusValue defines model for ImageStatusValue.
type ImageStatusValue string

// List of ImageStatusValue
const (
	ImageStatusValue_building    ImageStatusValue = "building"
	ImageStatusValue_failure     ImageStatusValue = "failure"
	ImageStatusValue_pending     ImageStatusValue = "pending"
	ImageStatusValue_registering ImageStatusValue = "registering"
	ImageStatusValue_success     ImageStatusValue = "success"
	ImageStatusValue_uploading   ImageStatusValue = "uploading"
)

// ImageTypes defines model for ImageTypes.
type ImageTypes string

// List of ImageTypes
const (
	ImageTypes_aws            ImageTypes = "aws"
	ImageTypes_azure          ImageTypes = "azure"
	ImageTypes_edge_commit    ImageTypes = "edge-commit"
	ImageTypes_edge_installer ImageTypes = "edge-installer"
	ImageTypes_gcp            ImageTypes = "gcp"
)

// List defines model for List.
type List struct {
	Kind  string `json:"kind"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
	Total int    `json:"total"`
}

// OSTree defines model for OSTree.
type OSTree struct {
	Ref *string `json:"ref,omitempty"`
	Url *string `json:"url,omitempty"`
}

// ObjectReference defines model for ObjectReference.
type ObjectReference struct {
	Href string `json:"href"`
	Id   string `json:"id"`
	Kind string `json:"kind"`
}

// PackageMetadata defines model for PackageMetadata.
type PackageMetadata struct {
	Arch      string  `json:"arch"`
	Epoch     *string `json:"epoch,omitempty"`
	Name      string  `json:"name"`
	Release   string  `json:"release"`
	Sigmd5    string  `json:"sigmd5"`
	Signature *string `json:"signature,omitempty"`
	Type      string  `json:"type"`
	Version   string  `json:"version"`
}

// Repository defines model for Repository.
type Repository struct {
	Baseurl    *string `json:"baseurl,omitempty"`
	Metalink   *string `json:"metalink,omitempty"`
	Mirrorlist *string `json:"mirrorlist,omitempty"`
	Rhsm       bool    `json:"rhsm"`
}

// Subscription defines model for Subscription.
type Subscription struct {
	ActivationKey string `json:"activation_key"`
	BaseUrl       string `json:"base_url"`
	Insights      bool   `json:"insights"`
	Organization  string `json:"organization"`
	ServerUrl     string `json:"server_url"`
}

// UploadOptions defines model for UploadOptions.
type UploadOptions interface{}

// UploadStatus defines model for UploadStatus.
type UploadStatus struct {
	Options interface{} `json:"options"`
	Status  string      `json:"status"`
	Type    UploadTypes `json:"type"`
}

// UploadTypes defines model for UploadTypes.
type UploadTypes string

// List of UploadTypes
const (
	UploadTypes_aws    UploadTypes = "aws"
	UploadTypes_aws_s3 UploadTypes = "aws.s3"
	UploadTypes_azure  UploadTypes = "azure"
	UploadTypes_gcp    UploadTypes = "gcp"
)

// User defines model for User.
type User struct {
	// Embedded struct due to allOf(#/components/schemas/ObjectReference)
	ObjectReference
	// Embedded fields due to inline allOf schema
	Groups *[]string `json:"groups,omitempty"`
	Key    *string   `json:"key,omitempty"`
	Name   string    `json:"name"`
}

// Page defines model for page.
type Page string

// Size defines model for size.
type Size string

// PostComposeJSONBody defines parameters for PostCompose.
type PostComposeJSONBody ComposeRequest

// GetErrorListParams defines parameters for GetErrorList.
type GetErrorListParams struct {

	// Page index
	Page *Page `json:"page,omitempty"`

	// Number of items in each page
	Size *Size `json:"size,omitempty"`
}

// PostComposeRequestBody defines body for PostCompose for application/json ContentType.
type PostComposeJSONRequestBody PostComposeJSONBody

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Create compose
	// (POST /compose)
	PostCompose(ctx echo.Context) error
	// The status of a compose
	// (GET /composes/{id})
	GetComposeStatus(ctx echo.Context, id string) error
	// Get the metadata for a compose.
	// (GET /composes/{id}/metadata)
	GetComposeMetadata(ctx echo.Context, id string) error
	// Get a list of all possible errors
	// (GET /errors)
	GetErrorList(ctx echo.Context, params GetErrorListParams) error
	// Get error description
	// (GET /errors/{id})
	GetError(ctx echo.Context, id string) error
	// Get the openapi spec in json format
	// (GET /openapi)
	GetOpenapi(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// PostCompose converts echo context to params.
func (w *ServerInterfaceWrapper) PostCompose(ctx echo.Context) error {
	var err error

	ctx.Set("Bearer.Scopes", []string{""})

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.PostCompose(ctx)
	return err
}

// GetComposeStatus converts echo context to params.
func (w *ServerInterfaceWrapper) GetComposeStatus(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "id" -------------
	var id string

	err = runtime.BindStyledParameter("simple", false, "id", ctx.Param("id"), &id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter id: %s", err))
	}

	ctx.Set("Bearer.Scopes", []string{""})

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetComposeStatus(ctx, id)
	return err
}

// GetComposeMetadata converts echo context to params.
func (w *ServerInterfaceWrapper) GetComposeMetadata(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "id" -------------
	var id string

	err = runtime.BindStyledParameter("simple", false, "id", ctx.Param("id"), &id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter id: %s", err))
	}

	ctx.Set("Bearer.Scopes", []string{""})

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetComposeMetadata(ctx, id)
	return err
}

// GetErrorList converts echo context to params.
func (w *ServerInterfaceWrapper) GetErrorList(ctx echo.Context) error {
	var err error

	ctx.Set("Bearer.Scopes", []string{""})

	// Parameter object where we will unmarshal all parameters from the context
	var params GetErrorListParams
	// ------------- Optional query parameter "page" -------------

	err = runtime.BindQueryParameter("form", true, false, "page", ctx.QueryParams(), &params.Page)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter page: %s", err))
	}

	// ------------- Optional query parameter "size" -------------

	err = runtime.BindQueryParameter("form", true, false, "size", ctx.QueryParams(), &params.Size)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter size: %s", err))
	}

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetErrorList(ctx, params)
	return err
}

// GetError converts echo context to params.
func (w *ServerInterfaceWrapper) GetError(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "id" -------------
	var id string

	err = runtime.BindStyledParameter("simple", false, "id", ctx.Param("id"), &id)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter id: %s", err))
	}

	ctx.Set("Bearer.Scopes", []string{""})

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetError(ctx, id)
	return err
}

// GetOpenapi converts echo context to params.
func (w *ServerInterfaceWrapper) GetOpenapi(ctx echo.Context) error {
	var err error

	ctx.Set("Bearer.Scopes", []string{""})

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.GetOpenapi(ctx)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.POST("/compose", wrapper.PostCompose)
	router.GET("/composes/:id", wrapper.GetComposeStatus)
	router.GET("/composes/:id/metadata", wrapper.GetComposeMetadata)
	router.GET("/errors", wrapper.GetErrorList)
	router.GET("/errors/:id", wrapper.GetError)
	router.GET("/openapi", wrapper.GetOpenapi)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file.
func GetSwagger() (*openapi3.Swagger, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error loading Swagger: %s", err)
	}
	return swagger, nil
}
