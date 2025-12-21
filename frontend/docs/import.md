# Importing Packages in FoLang

In FoLang, importing code from third-party libraries or your own libraries is done explicitly using the  
`@co.ddap.import` directive.

FoLang follows a **strict, explicit import model**:
- One file = one module
- No wildcard imports
- No directory imports
- No implicit inclusion

Everything that is used must be explicitly imported.

---

## Import Directive

```folang
@co.ddap.import(path="", package="", realm="", as="")
```

---

## Directive Fields

### 1. `path` — Canonical Module Path

`path` is a **canonicalized path** that resolves to **exactly one `.fol` file**.

Examples:

- `github.com/x/y` → `y.fol`
- `abc/bbc/ab` → `ab.fol`
- `urn:folang:co/ddap/foo` → `foo.fol`

Rules:

- One file = one module
- No directory imports
- No wildcard (`*`) or regex imports
- If a path is not imported, the file is not compiled or visible

---

### 2. `package` — Declared Package (Namespace)

`package` is the **package declared inside the referenced file**.

Examples:

```
Utils
Service
Proxy
Dto
```

Rules:

- Must exactly match the package declared in the `.fol` file
- Packages are **namespaces only**
- Packages do **not** control loading or inclusion
- Multiple modules may declare the same package

---

### 3. `realm` — Isolation Boundary

`realm` defines an explicit isolation domain.

Rules:

- Same `path` and `package` imported into different realms are treated as **different module instances**
- Default realm is `main` if not specified
- Realms are intended for:
  - third-party libraries
  - plugins
  - version coexistence

Realms **should not be used for normal application structuring**.

---

### 4. `as` — Mandatory Domain / Capability Alias

`as` is **mandatory** and defines a **domain or capability umbrella** under which imported modules are accessed.

Rules:

- `as` is a valid **qualified identifier**
- It represents a **business domain or capability**, not a physical path
- Multiple modules may share the same `as` value if they belong to the same domain

All imported symbols are accessed using:

```
<as>.<package>.<symbol>
```

---

## Why We Need `as`

Consider an application with multiple modules defining the same package and symbol names for different domains.

### Example Modules

1. `/myapp/accounts/User.fol`
2. `/myapp/hr/User.fol`

Both files declare:

- `package = "dto"`
- symbol: `Employee`

### Imports

```folang
@co.ddap.import(path="/myapp/accounts/User", package="dto", realm="main", as="accounts")
@co.ddap.import(path="/myapp/hr/User",       package="dto", realm="main", as="hr")
```

### Usage

```folang
accounts.dto.Employee
hr.dto.Employee
```

This avoids collisions while keeping business intent explicit.

---

## Grouping by Business Domain or Capability

Multiple modules belonging to the same business domain or capability may share the same `as` value.

### Example (HR domain)

```folang
@co.ddap.import(path="/myapp/hr/User",     package="dto",     realm="main", as="hr")
@co.ddap.import(path="/myapp/hr/Employee", package="service", realm="main", as="hr")
```

### Usage

```folang
hr.dto.Employee
hr.service.EmployeeServiceImpl
```

Here:

- `hr` represents the **HR business domain**
- `dto` and `service` represent different capabilities within that domain

---

## Notes

### Packages Do Not Imply Inclusion

- Only explicitly imported `path`s contribute symbols
- Sharing a package name does **not** load or link code automatically

---

### No Wildcard or Regex Imports

FoLang **strictly forbids**:

- `*` imports
- directory imports
- regex-based imports

This avoids:
- accidental dependency inclusion
- non-deterministic builds
- hidden coupling

Tooling may generate explicit imports, but the language itself requires them to be written explicitly.

---

### Realms Are for Isolation, Not Organization

- Realms isolate third-party libraries and plugins
- Business domains and capabilities are expressed via `as`, not via realms

---

