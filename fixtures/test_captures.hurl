GET {{url}}/pet/findByStatus?status=sold
HTTP 200
[Captures]
id: jsonpath "$[0].id"

GET {{url}}/pet/{{id}}
HTTP 200
[Captures]
name: jsonpath "$.name"

PUT {{url}}/pet
{

}
