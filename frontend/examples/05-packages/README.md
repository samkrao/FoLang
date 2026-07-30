# 05 — Packages

Spec: *Packages*, *Package in detail*, *Package Source Files*,
*Application Entry File*, *Package Access Rules*, *Package Aliasing*.

## Identity

A subfolder containing `.fol` files **is** a package. Dot paths start at
subfolders and the project root is never a package, so its name never appears
in any dot path.

```
05-packages/                      ←  application root, not a package
├── app.fol                       ←  entry file, not a package
├── hr/                           package "hr"
│   ├── employee/                 package "hr.employee"
│   │   ├── Employee.fol          the struct — pure data
│   │   ├── Employee.unit.fol     its companion unit — all behaviour
│   │   └── EmpService.fol        a standalone unit
│   ├── payroll/                  package "hr.payroll"
│   │   └── PayrollCalc.fol
│   └── empl/                     package "hr.empl", renamed to "hr.emp"
│       ├── package.fol           PLANNED — package alias declaration
│       └── Contractor.fol
└── auth/                         package "auth"
    └── Access.fol                the four visibility levels
```

## Rules these files demonstrate

- Multiple `.fol` files in one folder automatically belong to the same package.
- An ordinary package source file holds **exactly one** primary declaration.
  Directives and annotations before it do not count as a second one.
- Free functions never float at package-file scope — they are enclosed in a
  `co.lang.unit`.
- Variables, executable statements, explicit package declarations, and project
  or library metadata are forbidden at package-file scope.
- `_` in the declaration-name position derives the name from the filename.
  `Name.unit.fol` drops both `.fol` and the matching kind suffix, so
  `Employee.unit.fol` declares `Employee`.
- The entry file may depend on packages; packages may never depend on the
  entry file.

## Visibility

| Annotation | Same package | Parent | Sub-package | Unrelated | Entry / surface |
|---|---|---|---|---|---|
| `@co.dap.public` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `@co.dap.package` | ✅ | ✅ | ✅ | ❌ | ❌ |
| `@co.dap.protected` | ✅ | ❌ | ✅ | ❌ | ❌ |
| `@co.dap.private` | ✅ | ❌ | ❌ | ❌ | ❌ |
