# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [models_gateguard.proto](#models_gateguard.proto)
    - [User](#gateguard.models.User)
    - [Users](#gateguard.models.Users)
  
    - [UserStatus](#gateguard.models.UserStatus)
  
  
  

- [service_gateguard.proto](#service_gateguard.proto)
    - [EmailRequest](#gateguard.service.EmailRequest)
    - [Empty](#gateguard.service.Empty)
    - [TokenRequest](#gateguard.service.TokenRequest)
    - [TokenResponse](#gateguard.service.TokenResponse)
    - [UUIDRequest](#gateguard.service.UUIDRequest)
  
  
  
    - [GateguardService](#gateguard.service.GateguardService)
  

- [Scalar Value Types](#scalar-value-types)



<a name="models_gateguard.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## models_gateguard.proto



<a name="gateguard.models.User"></a>

### User
User is


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uuid | [bytes](#bytes) |  |  |
| email | [string](#string) |  |  |
| name | [string](#string) |  |  |
| avatar | [string](#string) |  |  |
| status | [UserStatus](#gateguard.models.UserStatus) |  |  |
| created | [int64](#int64) |  |  |






<a name="gateguard.models.Users"></a>

### Users
Users is


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#gateguard.models.User) | repeated |  |





 


<a name="gateguard.models.UserStatus"></a>

### UserStatus
UserStatus is enum with available User statuses

| Name | Number | Description |
| ---- | ------ | ----------- |
| UserUnknown | 0 | Unknown status |
| UserActive | 2 | Active user with all access to API |
| UserArchive | 3 | Archived user with no access to API |


 

 

 



<a name="service_gateguard.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## service_gateguard.proto



<a name="gateguard.service.EmailRequest"></a>

### EmailRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |






<a name="gateguard.service.Empty"></a>

### Empty
Empty is






<a name="gateguard.service.TokenRequest"></a>

### TokenRequest
TokenRequest is request message for methods
that needs only session JWT


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [bytes](#bytes) |  |  |






<a name="gateguard.service.TokenResponse"></a>

### TokenResponse
TokenResponse is response with session JWT


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [bytes](#bytes) |  |  |






<a name="gateguard.service.UUIDRequest"></a>

### UUIDRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uuid | [bytes](#bytes) |  |  |





 

 

 


<a name="gateguard.service.GateguardService"></a>

### GateguardService
GateguardService is GRPC server implementation
for gateguard requests

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SignInOAuth | [.gateguard.models.User](#gateguard.models.User) | [TokenResponse](#gateguard.service.TokenResponse) | SignInOAuth is |
| SignOut | [TokenRequest](#gateguard.service.TokenRequest) | [Empty](#gateguard.service.Empty) | SignOut is |
| CheckAuth | [TokenRequest](#gateguard.service.TokenRequest) | [.gateguard.models.User](#gateguard.models.User) | CheckAuth is |
| UserByUUID | [UUIDRequest](#gateguard.service.UUIDRequest) | [.gateguard.models.User](#gateguard.models.User) |  |
| UserByEmail | [EmailRequest](#gateguard.service.EmailRequest) | [.gateguard.models.User](#gateguard.models.User) |  |
| DeleteUser | [TokenRequest](#gateguard.service.TokenRequest) | [Empty](#gateguard.service.Empty) | DeleteUser is |
| Users | [Empty](#gateguard.service.Empty) | [.gateguard.models.Users](#gateguard.models.Users) | Users is |

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

