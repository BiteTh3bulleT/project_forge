# Security Test Matrix

| Area | Test | Expected |
|---|---|---|
| Auth | /health no token | 200 |
| Auth | /forge/models no token | 401 |
| Auth | valid token | allowed |
| Bind | 0.0.0.0 no token | startup fail |
| Docker | default env | no unauth wildcard |
| Approval | body actor only | reject |
| CORS | random localhost prod | reject |
| Context | /etc/passwd import | reject |
| Jobs | running job restart | recovered |
| Windows | proc tools | no bash/kill assumption |
