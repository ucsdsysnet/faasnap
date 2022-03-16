# swagger_client.DefaultApi

All URIs are relative to *http://localhost:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**functions_get**](DefaultApi.md#functions_get) | **GET** /functions | 
[**functions_post**](DefaultApi.md#functions_post) | **POST** /functions | 
[**invocations_post**](DefaultApi.md#invocations_post) | **POST** /invocations | 
[**metrics_get**](DefaultApi.md#metrics_get) | **GET** /metrics | 
[**net_ifaces_namespace_put**](DefaultApi.md#net_ifaces_namespace_put) | **PUT** /net-ifaces/{namespace} | 
[**snapshots_post**](DefaultApi.md#snapshots_post) | **POST** /snapshots | 
[**snapshots_put**](DefaultApi.md#snapshots_put) | **PUT** /snapshots | 
[**snapshots_ss_id_mincore_get**](DefaultApi.md#snapshots_ss_id_mincore_get) | **GET** /snapshots/{ssId}/mincore | 
[**snapshots_ss_id_mincore_patch**](DefaultApi.md#snapshots_ss_id_mincore_patch) | **PATCH** /snapshots/{ssId}/mincore | 
[**snapshots_ss_id_mincore_post**](DefaultApi.md#snapshots_ss_id_mincore_post) | **POST** /snapshots/{ssId}/mincore | 
[**snapshots_ss_id_mincore_put**](DefaultApi.md#snapshots_ss_id_mincore_put) | **PUT** /snapshots/{ssId}/mincore | 
[**snapshots_ss_id_patch**](DefaultApi.md#snapshots_ss_id_patch) | **PATCH** /snapshots/{ssId} | 
[**snapshots_ss_id_reap_delete**](DefaultApi.md#snapshots_ss_id_reap_delete) | **DELETE** /snapshots/{ssId}/reap | 
[**snapshots_ss_id_reap_get**](DefaultApi.md#snapshots_ss_id_reap_get) | **GET** /snapshots/{ssId}/reap | 
[**snapshots_ss_id_reap_patch**](DefaultApi.md#snapshots_ss_id_reap_patch) | **PATCH** /snapshots/{ssId}/reap | 
[**ui_data_get**](DefaultApi.md#ui_data_get) | **GET** /ui/data | 
[**ui_get**](DefaultApi.md#ui_get) | **GET** /ui | 
[**vmms_post**](DefaultApi.md#vmms_post) | **POST** /vmms | 
[**vms_get**](DefaultApi.md#vms_get) | **GET** /vms | 
[**vms_post**](DefaultApi.md#vms_post) | **POST** /vms | 
[**vms_vm_id_delete**](DefaultApi.md#vms_vm_id_delete) | **DELETE** /vms/{vmId} | 
[**vms_vm_id_get**](DefaultApi.md#vms_vm_id_get) | **GET** /vms/{vmId} | 


# **functions_get**
> list[Function] functions_get()



Return a list of functions

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()

try:
    api_response = api_instance.functions_get()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->functions_get: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**list[Function]**](Function.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **functions_post**
> functions_post(function=function)



Create a new function

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
function = swagger_client.Function() # Function |  (optional)

try:
    api_instance.functions_post(function=function)
except ApiException as e:
    print("Exception when calling DefaultApi->functions_post: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **function** | [**Function**](Function.md)|  | [optional] 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **invocations_post**
> object invocations_post(invocation=invocation)



Post an invocation

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
invocation = swagger_client.Invocation() # Invocation |  (optional)

try:
    api_response = api_instance.invocations_post(invocation=invocation)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->invocations_post: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **invocation** | [**Invocation**](Invocation.md)|  | [optional] 

### Return type

**object**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **metrics_get**
> metrics_get()



Metrics

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()

try:
    api_instance.metrics_get()
except ApiException as e:
    print("Exception when calling DefaultApi->metrics_get: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **net_ifaces_namespace_put**
> net_ifaces_namespace_put(namespace, interface=interface)



Put a vm network

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
namespace = 'namespace_example' # str | 
interface = swagger_client.Interface() # Interface |  (optional)

try:
    api_instance.net_ifaces_namespace_put(namespace, interface=interface)
except ApiException as e:
    print("Exception when calling DefaultApi->net_ifaces_namespace_put: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **namespace** | **str**|  | 
 **interface** | [**Interface**](.md)|  | [optional] 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_post**
> Snapshot snapshots_post(snapshot=snapshot)



Take a snapshot

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
snapshot = swagger_client.Snapshot() # Snapshot |  (optional)

try:
    api_response = api_instance.snapshots_post(snapshot=snapshot)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_post: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **snapshot** | [**Snapshot**](Snapshot.md)|  | [optional] 

### Return type

[**Snapshot**](Snapshot.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_put**
> Snapshot snapshots_put(from_snapshot, mem_file_path)



Put snapshot (copy)

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
from_snapshot = 'from_snapshot_example' # str | 
mem_file_path = 'mem_file_path_example' # str | 

try:
    api_response = api_instance.snapshots_put(from_snapshot, mem_file_path)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_put: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **from_snapshot** | **str**|  | 
 **mem_file_path** | **str**|  | 

### Return type

[**Snapshot**](Snapshot.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_mincore_get**
> object snapshots_ss_id_mincore_get(ss_id)



Get mincore state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 

try:
    api_response = api_instance.snapshots_ss_id_mincore_get(ss_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_mincore_get: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 

### Return type

**object**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_mincore_patch**
> snapshots_ss_id_mincore_patch(ss_id, state)



Change mincore state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 
state = swagger_client.State() # State | 

try:
    api_instance.snapshots_ss_id_mincore_patch(ss_id, state)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_mincore_patch: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 
 **state** | [**State**](.md)|  | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_mincore_post**
> snapshots_ss_id_mincore_post(ss_id, layer)



Add mincore layer

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 
layer = swagger_client.Layer() # Layer | 

try:
    api_instance.snapshots_ss_id_mincore_post(ss_id, layer)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_mincore_post: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 
 **layer** | [**Layer**](.md)|  | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_mincore_put**
> snapshots_ss_id_mincore_put(ss_id, source=source)



Put mincore state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 
source = 'source_example' # str |  (optional)

try:
    api_instance.snapshots_ss_id_mincore_put(ss_id, source=source)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_mincore_put: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 
 **source** | **str**|  | [optional] 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_patch**
> snapshots_ss_id_patch(ss_id, state=state)



Change snapshot state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 
state = swagger_client.State() # State |  (optional)

try:
    api_instance.snapshots_ss_id_patch(ss_id, state=state)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_patch: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 
 **state** | [**State**](.md)|  | [optional] 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_reap_delete**
> snapshots_ss_id_reap_delete(ss_id)



delete reap state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 

try:
    api_instance.snapshots_ss_id_reap_delete(ss_id)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_reap_delete: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_reap_get**
> snapshots_ss_id_reap_get(ss_id)



get reap state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 

try:
    api_instance.snapshots_ss_id_reap_get(ss_id)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_reap_get: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **snapshots_ss_id_reap_patch**
> snapshots_ss_id_reap_patch(ss_id, cache=cache)



Change reap state

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
ss_id = 'ss_id_example' # str | 
cache = true # bool |  (optional)

try:
    api_instance.snapshots_ss_id_reap_patch(ss_id, cache=cache)
except ApiException as e:
    print("Exception when calling DefaultApi->snapshots_ss_id_reap_patch: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ss_id** | **str**|  | 
 **cache** | **bool**|  | [optional] 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ui_data_get**
> ui_data_get()



UI

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()

try:
    api_instance.ui_data_get()
except ApiException as e:
    print("Exception when calling DefaultApi->ui_data_get: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ui_get**
> ui_get()



UI

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()

try:
    api_instance.ui_get()
except ApiException as e:
    print("Exception when calling DefaultApi->ui_get: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **vmms_post**
> VM vmms_post(vmm=vmm)



Create a VMM in the pool

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
vmm = swagger_client.VMM() # VMM |  (optional)

try:
    api_response = api_instance.vmms_post(vmm=vmm)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->vmms_post: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **vmm** | [**VMM**](.md)|  | [optional] 

### Return type

[**VM**](VM.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **vms_get**
> list[VM] vms_get()



Returns a list of active VMs

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()

try:
    api_response = api_instance.vms_get()
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->vms_get: %s\n" % e)
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**list[VM]**](VM.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **vms_post**
> VM vms_post(vm=vm)



Create a new VM

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
vm = swagger_client.VM() # VM |  (optional)

try:
    api_response = api_instance.vms_post(vm=vm)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->vms_post: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **vm** | [**VM**](.md)|  | [optional] 

### Return type

[**VM**](VM.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **vms_vm_id_delete**
> vms_vm_id_delete(vm_id)



Stop a VM

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
vm_id = 'vm_id_example' # str | 

try:
    api_instance.vms_vm_id_delete(vm_id)
except ApiException as e:
    print("Exception when calling DefaultApi->vms_vm_id_delete: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **vm_id** | **str**|  | 

### Return type

void (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **vms_vm_id_get**
> VM vms_vm_id_get(vm_id)



Describe a VM

### Example
```python
from __future__ import print_function
import time
import swagger_client
from swagger_client.rest import ApiException
from pprint import pprint

# create an instance of the API class
api_instance = swagger_client.DefaultApi()
vm_id = 'vm_id_example' # str | 

try:
    api_response = api_instance.vms_vm_id_get(vm_id)
    pprint(api_response)
except ApiException as e:
    print("Exception when calling DefaultApi->vms_vm_id_get: %s\n" % e)
```

### Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **vm_id** | **str**|  | 

### Return type

[**VM**](VM.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

