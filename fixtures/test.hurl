POST {{url}}/people
{
      "name": "Test Name",
      "height": "172",
      "mass": "77",
      "birth_year": "1999"
}
HTTP/1.1 201
[Captures]
id: jsonpath "$.id"
