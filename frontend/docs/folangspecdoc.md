
###### arithmatic operators: +, -, *, /, %, **, ++, --
###### logical operators: &&, ||, !,&, |
###### comparison operators: ==, !=, <, >, <=, >=

###### Othe operators
     
     @, #, !, ~, $, ^, (, ), _, `, ?, {, [, ], }, \, :, ;, ", ', =, ., ?=, :=, ::=, ,, .., ..., <.., ..<, <..<, =>>, =>, ->, <-, ->>, <->,


###### other reserved operators 

    λ	⒪ , â,  Ť,  ∀,  ∃,  ○,  ö, ∪, Ṡ,  Ŝ, ṁ, 𝚷, ⇛ , 𝑓 , 𝒯 , 𝘷 , 𝓕 , ↓, λ, ∂, or ⊥ ↧ or ⇓
   

###### Reserved words

    co, let, this, impl,self,for,forall

   ###### Difference between this and self

        this is for instances and objects 
        self is for classes
        static there is no shortcut they can be on variable or classname

        otherwise self and this can acess member variables

#### Variable declaration

    someVar co.lang.int;
    someString co.lang.string;

#### Variable declaration with initialization

    someBool co.lang.bool = co.const.true;
    someInt co.lang.int = 42;

#### Variable declaration with type inference

    someVal := "Hello, World!";
    someNum := 3.14;  // if not define  define and initialize else throws error
    someR ?= "Kamesh";//if not define define and initialize else assign value

#### pointer declaration

    somePtr co.lang.int->(*);
    someDblPtr co.lang.int->(**);

#### Array declaration

    someArray co.lang.int ->([5]);
    someDblArray co.lang.int->([2,3]);
    someJaggedArray co.lang.int->([2][3]);
    someVLArray co.lang.int->([...]);
    someZeroLA   co.lang.int->([0]);    
    someZeroDimA  co.lang.int->([.]);

#### Array declaration with initialization
    
    someInitializedArray co.lang.int ->([3]) = [1, 2, 3];
    someInitializedArray1 co.lang.int ->([]) = [1, 2, 3];
    someInitializedDblArray co.lang.int->([,]) = [[1, 2], [3, 4]];

#### Reference Declaration

    someRef co.lang.int ->(&);

#### LValue reference declaration

    someLValueRef co.lang.int ->(&&);

#### Heap allocated Reference declaration

    someHpRef co.lang.int->(~);

#### address declaration

    someAddr co.lang.int ->(@);

#### Thunk Declaration

    someThunk co.lang.int ->(^);

#### slice declaration
    
    someSlice co.lang.int ->([:]);

#### range declaration

    // Typed range variable declaration (produces RangeVariableDeclStmt):
    someRange co.lang.int->(..);

    // Inferred range declarations (produce VarDeclarationStmt with RangeExpr value):
    // RangeExpr fields: Lower, Upper, ExcludeStart, ExcludeEnd
    rangeI := 1..10;       // [1, 10]  — token: ..   (DOT_DOT),     ExcludeStart=false, ExcludeEnd=false
    rangeS := 0<..5;       // (0, 5]   — token: <..  (LT_DOT_DOT),  ExcludeStart=true,  ExcludeEnd=false
    rangeL := 0..<100;     // [0, 100) — token: ..< (DOT_DOT_LT),  ExcludeStart=false, ExcludeEnd=true
    rangeB := 0<..<100;    // (0, 100) — token: <..< (LT_DOT_DOT_LT), ExcludeStart=true, ExcludeEnd=true
    rangeE := ..100;       // open lower bound  (_, 100] — token: .. prefix (Lower=nil)
    rangeF := 1..;         // open upper bound  [1, _)  — token: .. infix  (Upper=nil)

#### auto and dynamic variable declaration

    someAutoVar co.lang.auto = "Hello"; //auto needs initialization as type inferred from vallue
    someDynamicVar co.lang.dynamic; // it is like dynamic typing 

#### Values

    someVar co.lang.data = 10; //initialization required

#### Fat pointers
 
    x co.lang.int->(*, kind="", meta={});

      define x co.lang.int->(*, meta={});

      define y co.lang.int->(*, meta={len:co.lang.usize,vtab:somepkg.VTable->(*)})

      define z co.lang.int->(*,kind=region, meta={})

           Pointer
           ├── base_type: T
           ├── kind: <FatKind>
           │    ├── thin
           │    ├── slice
           |    |── relative
           │    ├── trait
           │    ├── buffer
           │    ├── view
           │    ├── opaque
           │    ├── custom
           │    └── (region)  ← optional syntactic sugar
           └── meta:
                ├── region: heap | stack | global | numa(N) | mmio | constant | …
                ├── len, cap, vtab, bits, endian, …

#### Integerpointers etc,

        a. Signed
        
            define y co.lang.intptr;

        b. Unsigned

            define z co.lang.uintptr; 
        
        c. Diff

            define p co.lang.ptrdiff;
    
#### Relative Pointers

        a. define z co.lang.int->(*,kind=relative, meta={})

#### Package declaration
 
    mypackage co.lang.package={

    }


#### struct declaration
    
    myStruct co.lang.struct={
        field1 co.lang.int;
        field2 co.lang.string;
        field3 co.lang.bool;
    }

#### enum declaration
    
    myEnum co.lang.enum={
        Variant1,
        Variant2,
        Variant3
    }

#### union declaration
    
    myUnion co.lang.union={
        intValue co.lang.int;
        strValue co.lang.string;
    }   

#### Module declaration
    
    myModule co.lang.signature={
        // module contents
    }


    @co.dap.modulesig(myModule)           )
    impl mymod co.lang.module->(matches=myModule) = {


    }

    mm myModule= mymod;

#### Functions
   
   ##### Normal
    
        fun1 (k co.lang.int, b co.lang.char)->(co.lang.int, co.lang.char)={
            // function body
        }

   ##### curried
   
        add(first co.lang.int)(second co.lang.int)->(co.lang.int)={
            first + second
        }

   ##### closure

        adder() -> ((co.lang.int) -> co.lang.int) ={
            sum co.lang.int = 0
            this.return  (x co.lang.int) -> (co.lang.int) = {
                sum += x
                this.return sum
            }
        }   

   ##### Functions taking functions as parameters and returning functions as parameters

       Syntax1:  Passing function syntax and return function syntax
       
            someFunction (r (co.lang.int, co.lang.int)->(co.lang.int))->((co.lang.int)->(co.lang.int))={}
       
       Syntax2:  Passing a function type and return function type

            someFArg co.lang.type = (co.lang.int, co.lang.int)->(co.lang.int)
            someFRet co.lang.type = (co.lang.int)->(co.lang.int)
            
            someFunction (r someFArg)->(someFRet)={}
       
       Syntax3: Passing function objects and return function objects

            someFArg co.lang.function = (a co.lang.int, b co.lang.int)    -> (co.lang.int)={
                this.return a + b;
            }

            someFRet co.lang.function = (a co.lang.int)    -> (co.lang.int)={
                this.return a * 2;
            }

            someFunction (r someFArg)->(someFRet)={}


   ##### Anonymous Functions and objects

   ##### anyonymous classes/types

        emp := co.lang.class{

        };

        empObj := emp.new();

        empobj1 := co.lang.class{
            name string
        }.new();

   ##### anonymous functions

            add := (a int, b int) -> (int) {
                this.return a + b;
            };

            res := (a int, b int) -> (int) {
                this.return a * b;
            })(10, 20);

   ##### lambda

        Only allowed as an inline callback argument to collection operations including arrays (e.g. map, filter, reduce, forEach, sortBy, groupBy, etc.). Using |...| anywhere else is a syntax/lint error.

        syntax: |x,y|=>  x+y ;


   ###### collection use — allowed
   
        nums.map(|x| => x*x)
        words.filter(|s| => s.len() > 3)
        pairs.reduce(|acc, e| => acc + e, 0)
        dict.map(|k, v| => v * 10)
        list.sortBy(|a, b| => a.score - b.score)  // typical comparator callback

   ###### inner function

        myfun(a co.lang.int, b co.lang.int)->(co.lang.int)={
            p co.lang.int = 10; 
            someother()->()={
                co.out.println(p);
            }
            someother();
            p =20;
            someother();
        }

   ###### function objects

        myobj co.lang.function =  (a co.lang.int, b co.lang.int)-> (co.lang.int)={
        this.return a + b;
        }

        add (a co.lang.int, b co.lang.int)->(co.lang.int){ this.return a + b; }
        oObj co.lang.function = add;

        funtype co.lang.type = (a co.lang.int, b co.lang.int)->(co.lang.int);

        closure(factor int) => (x int)= x * factor; 

        curry(factor int)(val int)= factory * val;

       
   ##### Default parameters
    
        fun1 (k co.lang.int, b co.lang.char = 10)->(co.lang.int, co.lang.char)={
        }

   ##### Variadic functions curried functions are not allowed to be variadic, and vice versa.

        fun1 (k co.lang.int, ...b co.lang.char)->(co.lang.int, co.lang.char)={
        }

   ##### Optional params

        fun1(k? co.lang.int)->()={
            if k.omitted{

            }else{
                    
            }
        }
    
   ##### Named parameters
    
        fun1(~k co.lang.int)->()={
            
        } 

   ##### function delegates
   
        @co.dap.delegate someDelegate co.lang.delegate = (a co.lang.int, b co.lang.int) -> (co.lang.int, co.lang.int );

   ##### function chaining

        fetchEmployee(empId co.lang.string )->(Employee)=>>empMod.getEmploee(this,empId);

   ##### associated functions
   
        (emp empMod.Employee) fetchEmployee(empId co.lang.string)->(empMod.Employee)=>>empMod.getEmployee(emp, empId);

   ##### Generic

                
        @co.dap.generic(at="runtime",refied=true,where="callsite",impredicative=false)
        
        where
         
           1. usesite
           2. callsite/declaration site

        at

          1. runtime
          2. compiletime acts like C++ templates

        refied

           1. true
           2. false
        
        typename/type

            is a dictionary contains attributes

            1. variance

                a. covariant
                b. invariant
                c. contravariant

            2. bound

                is the type to bind

            3. kind

                a. param
                b. result
                c. var
                d. arg

            4. default
            5. nullable
            6. inclusive
            7. types

                list of allowed types for constraints

        
        @co.dap.generic(
            at=runtime,
            type={
                T: {variance:invariant, bound=Number,Kind:Param},
                R: {variance:invariant, bound=Number,Kind:Return}
            }   
        )
        add ( a T, b T) ->(R)= { this.return a + b ;}



   ##### indexer

        MyList co.lang.struct ={
            eles co.lang.int ->([*]);
        }

        @co.dap.indexer (symbol="[]")
        (g MyList) get (index co.lang.int)->(co.lang.int) ={
            this.return g.eles[index]
        }

        @co.dap.indexer(symbol="[]=")
        (g MyList) set (index co.lang.int, value co.lang.int) -> () ={
            g.ele[index] = value
        }

#### Templates

   ##### typed

        @co.dap.template
        add(a co.lang.int, b co.lang.int)->(co.lang.int) ={
            this.return a +b;

        }

   ##### untyped
   
        @co.dap.template
        add(a,b)->(co.lang.untyped) ={
            this.return a+b;
        }

        @co.dap.template 
        add(a co.lang.int, b co.lang.int)->(co.lang.int) ={
            this.return a +b;

        }


#### Macros
       a.

        @co.dap.macro define say()->()={ this.return co.macro.quote({ println("Line 1") println("Line 2") }); }

       
        b.
       
            @co.dap.macro
            yes_esc_assign()->(co.lang.untyped)={
                this.return co.macro.quote({
                    co.macro.esc(y) = 42
                    println("Inside macro: y = ", y)
                });
            }


        c.

            @co.dap.macro
            define debug(expr)->(co.lang.untyped)={
                let tmp = co.macro.gensym(co.lang.var,"tmp")   
                                    //conflict with local variables after code generration
                this.return co.macro.quote({
                    tmp = co.macro.esc(expr)
                    println("Result: ", tmp)
                    tmp
                });
            }

        d.

            if else condition macro
        
            @co.dap.macro(
                group = {items:["if","else"],chain:true},
                sugarform={forms:["if expr block"]},
                bind={vars:["x"]},
                isolate={vars:["temp", "index"]},  // → require gensym for those
                gensym={prefix:"tmp_"},           //  → set gensym naming strategy
                hygienic=true,                    //  → opt-out of hygiene manually
                argtransform={param:"body", wrap:"lambda",whentype:"block"},
                desugar={exprs:["if($cond) { $block }" => "if($cond,$block)"]},
                mode="inject"                   // # or "call" or "meta" or "inline" or "template"  
            
            )
            define if ( condition expr, body block)->()={

            }

            define blockormacro co.lang.Kind=  block | macro
            
            @co.dap.macro(
                group= {items:["if","else"],chain:true},
                sugarform={forms:["else block","else if"]},
                chainswith={macro:"if", position:"immediate", required:true},
                argtransform={param:"body", wrap:"lambda",whentype:"block"},
                standalone=false,
                desugar={exprs:[
                "else if($cond) { $block }" => "else(if($cond, $block))",
                "else { $elseblock }" => "else($elseblock)"
                ]},
            )
            define else (body blockormacro)->()={

            }


        e.

            Others

            1. @co.dap.compose(using=["base_if", "blockify"])
            2. @co.dap.guard(expr="is_bool_expr(expr)")
            3. Quasiquote Macros use co.macro.quote and co.macro.unquote

#### Let 

   ##### Let bindings
   
        y co.lang.int = let({x =10}).in({x+1});
        y co.lang.int = let({$ =10}).in({$+1});  $ is a special identifier that can be used 
        in let bindings to refer to the value being defined, allowing for recursive definitions or self-referential expressions.

        x co.lang.int= (x+1).where(x=10);
        x co.lang.int=($ + 1).where($=10);

    

        let fib(0) = 1
        let fib(1) =  1
        let fib(n) = fib(n-1)+ fib(n-2)


#### import statement

    @co.ddap.import(path="", package="", realm="", parent-realm="", as="")


###### Directive Fields

###### 1. `path` — Canonical Module Path

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

###### 2. `package` — Declared Package (Namespace)

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

###### 3. `realm` — Isolation Boundary

`realm` defines an explicit isolation domain.

`realm` are hierarchial
       
       core  (folang core realm restricted)
         |
         |
         |
         |
       user defined 

Rules:

- Same `path` and `package` imported into different realms are treated as **different module instances**
- Default realm is `main` if not specified 
- Realms are intended for:
  - third-party libraries
  - plugins
  - version coexistence

Realms **should not be used for normal application structuring**.

---
###### 4. `parent-realm` — Mandatory Domain / Capability Alias
`parent-realm` defines associates hierarchy if not specified it will be defaulted to core.
`core` realm is the realm where folang core packages like co. are loaded
`core` realm is restricted and root of all realms  


###### 5. `as` — Mandatory Domain / Capability Alias

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

###### Why We Need `as`

Consider an application with multiple modules defining the same package and symbol names for different domains.

###### Example Modules

1. `/myapp/accounts/User.fol`
2. `/myapp/hr/User.fol`

Both files declare:

- `package = "dto"`
- symbol: `Employee`

###### Imports

```folang
@co.ddap.import(path="/myapp/accounts/User", package="dto", realm="main", as="accounts")
@co.ddap.import(path="/myapp/hr/User",       package="dto", realm="main", as="hr")
```

###### Usage

```folang
accounts.dto.Employee
hr.dto.Employee
```

This avoids collisions while keeping business intent explicit.

---

###### Grouping by Business Domain or Capability

Multiple modules belonging to the same business domain or capability may share the same `as` value.

###### Example (HR domain)

```folang
@co.ddap.import(path="/myapp/hr/User",     package="dto",     realm="main", as="hr")
@co.ddap.import(path="/myapp/hr/Employee", package="service", realm="main", as="hr")
```

###### Usage

```folang
hr.dto.Employee
hr.service.EmployeeServiceImpl
```

Here:

- `hr` represents the **HR business domain**
- `dto` and `service` represent different capabilities within that domain

---

###### Notes

###### Packages Do Not Imply Inclusion

- Only explicitly imported `path`s contribute symbols
- Sharing a package name does **not** load or link code automatically

---

###### No Wildcard or Regex Imports

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

###### Realms Are for Isolation, Not Organization

- Realms isolate third-party libraries and plugins
- Business domains and capabilities are expressed via `as`, not via realms

---


---

###### Alias–Realm Binding Rule

Within a single realm, a module identified by a given `path` **MUST NOT** be imported under more than one domain alias (`as`).

- The same module `path` may appear under the same or different aliases **only if the realm is different**.
- Within a realm, a module has **exactly one domain identity**.

This rule prevents semantic reclassification of the same code under multiple business domains and guarantees stable meaning within a realm.

**Invalid (same realm, same packages, same paths(modules), different aliases):**
```folang
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main", as="hr")
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main", as="accounts") // ERROR

```
**Invalid (same realm, same packages, same aliases, different paths (modules)):**
```folang
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main", as="hr")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto",realm="main",as="hr") //ERROR
```


**InValid (different realms):**

```folang
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main",    as="hr")
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="plugin1", as="accounts")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto",realm="pluginA",as="hr")  //ERROR
```
***Why InValid***

****Lets see by example:****

hr.dto which dto from  /myapp/hr/User under main realm or /myapp/v1/hr/User under pluginA realm ?

There are two ways to resolve


*****1. Valid (different realms):*****

```folang
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main",    as="hr")
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="plugin1", as="accounts")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto",realm="pluginA", parent-realm="main",as="hr") 
```

Now realm PluginA is child to main so when we say
hr.dto it always checks child if not present traverses to parent till core


           core
             |
             |
            / \  
        main  plugin1
          |
        pluginA

****Note:****
    
       Folang searches the symbols from all leaf nodes traversing to root that is `core` before complaining for not found.

       in the above example leaf nodes are pluginA and plugin1

       PluginA's hr shadows main's hr even though the code compiles hr always means pluginA's hr

       you may not want that then the second way as below

*****2. Valid (differrent realms) *****

```folang
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="main", as="hr")
@co.ddap.import(path="/myapp/hr/User", package="dto", realm="plugin1", as="accounts")
@co.ddap.import(path="/myapp/v1/hr/User", package="dto",realm="pluginA", as="v1.hr") 
```

now we can say 
   hr.dto
      or
   v1.hr.dto


                core
                  |
                  |
                / | \
               /  |  \
              /   |   \
             /    |    \
          main   plugin1 pluginA

  ***Note***

     For compiled binary these realms are simply where the symbols live
     
     When we annotate with `@co.dap.dynamicruntime` these acts like loaders where objects also reside.

     @co.dap.dynamicruntime annotation is not supported in release 1.0


#### annotations, directives, pragmas and decorators

    @co.dap.annotation
    myAnnotation () = {
        // annotation contents
    }

    @co.dap.directive
    myDirective () = {
        // directive contents
    }

    @co.dap.pragma
    myPragma () = {
        // pragma contents
    }

    @co.dap.decorator
    myDecorator () = {
        // decorator contents
    }



    dap co.lang.package={
  
        Variance co.lang.enum={
            covariant,
            contravariant,
            invariant
        }

        scope co.lang.enum={
            runtime,
            comile,
            }
  

        TypeParamSpec co.lang.struct= {
            Name string;
            variance Variance;
            bound string;
        }

     
        //Keyword level (attaching to define)
        @co.dap.annotation
        someAnn (...TypeParamSpec) = {
            
                scope scope = scope.runtime  // or "compile"
                process(ctx co.lang.annotationcontext)->()={}
                before()->()={}
                after()->()={}
                around()->()={}
                onError()->()={}

                
            }
        }


##### pattern matching

    x co.lang.int = 10;

    x.match.case(n: n > 10 => { n= n+100;"GT"}).case(_: n < 10 => "LT").default("EQ");
    x.match.case ((e: Employee) => ...). case ((_: Dept)     => ...)

    x.match(co.pattern.Type).case(co.lang.int   => ...).case(co.lang.float => ...);
    x.match(co.pattern.Value).case (0 => ...).case (1 => ...);
    x.match(co.pattern.Instance).case(xx.CAT=>...).case(xx.DOG => ...).default("Animal");
    x.match(co.pattern.Object).case(xx.Ball => "Ball").case(xx.CAT=> "CAT").default("Unknown");
   
   ###### Object vs Instance in folang

        Instance is from types of class/structs
        Objects are anything like functions, class, structs, types, etc..

    x.match(co.pattern.Shape).case (Point{x, y} => ...).default(_=> ...);

    x.match(co.pattern.Any).(case co.lang.int   => ...).case (co.lang.float => ...).case (0 => ...).default(_=> ...);
    or
    x.match.case(co.lang.int   => ...).case (co.lang.float => ...).case (0 => ...).default(_=> ...);
    x.match(PositiveEvenMatcher).case(0   =>  "Neither even nor odd" ).case(2   =>"First Even Prime").default(...);
    
    @co.dap.matcher
    Matcher(T) = {
        matchCase(value T, pattern co.lang.untyped)
            -> (co.lang.int, MatchBindings); 
            //int in return is number of matches 0 no match >0 match
    }
    
    @co.dap.instance(matcher=Matcher for=co.lang.int)
    impl PositiveEvenMatcher->(instance=Matcher(co.lang.int))  = {
        matchCase(value co.lang.int, pat co.lang.untyped)->(co.lang.int, MatchBindings) = {
        // user logic…
        }
    }

    // _ is a special discard/ wildcard variable usable only inside pattern matching/ contains/iterators constructs (and similar discardable constructs), and is not a normal variable name by alone elsewhere _ must accompany by some ASCII letter or number .


##### Monads, applicatives, functors Monoids and Transformers

   ##### Functors

        @co.dap.Functor
        Functor (F) ={
            map(value F(A), f (A)->B) -> (F(B));
        }

        @co.dap.instance(typeclass=Functor, for=List)
        impl _ ->(instance=Functor(List)) ={
            map(value List(A), f (A)->B) ->( List(B)) ={
                result = List(B){}

                value.each(_,item).do({
                    result.append(f(item))
                });

                this.return result
            }
        }
        @co.dap.instance(typeclass=Functor, for=Set)
            impl _ ->(instance=Functor(Set)) ={
            map(value Set(A), f (A)->B) ->( Set(B)) ={
                result = Set(B){}

                value.each(_,item).do({
                    result.append(f(item))
                });

                this.return result
            }
        }

   ##### Applicative

        @co.dap.applicative
        Applicative(F) ={
            pure(x A)-> (F(A));

            apply(fab F(A->B), fa F(A))-> (F(B));
        }

        @co.dap.instance(typeclass=Applicative, for=Option)
        impl _->(instance=Applicative(Option)) ={

            pure(x A) -> (Option(A)) ={
                this.return Some(x);
            }

            apply(fab: Option(A->B), fa: Option(A)) -> (Option(B))= {
                this.return (fab, fa)
                    .match
                    .case((Some(f), Some(x)) => Some(f(x)))
                    .default(None());


                
            }
        }

   ##### Monods

    
        @co.dap.monad
        Monad(F) ={
            pure(x A) -> (F(A));
            flatMap(fa F(A), f (A)->F(B)) -> (F(B));
        }
        @co.dap.instance(typeclass=Monad, for=Option)
        impl _->(insance=Monad(Option)) {
            pure(x A) -> (Option(A))= {
                this.return Some(x);
            }

            flatMap(fa Option(A), f (A)->Option(B)) -> (Option(B))= {
                this.return fa.match().case(Some(x) => f(x)).default(None);
                
            }
        }


   ##### Monoid

        @co.dap.monoid
        Monoid(T) ={
            empty() -> (T);
            combine(a T, b T) -> (T);
        }
        @co.dap.instance(typeclass=Monoid, for=co.lang.int)
        impl _-> (instance=Monoid(co.lang.int)) ={
            empty()->(co.lang.int) ={
                this.return 0;
            }

            combine(a co.lang.int, b co.lang.int) ->(co.lang.int) ={
                this.return a + b;
            }
        }
   

   ##### Transformer

        @co.dap.transformer
        Transformer(F(_), G(_)) ={
            map(value F(A), f (A)->B) -> (G(B));
        }   
        
        @co.dap.instance(typeclass=Transformer, for=[List,Set])
        impl _ -> (instance=Transformer(List(_), Set(_))) ={
            map(value List(A), f (A)->B) -> (Set(B)) ={
                result = Set(B) {}
                value.each(_,item).do({
                    result.insert(f(item))     
                });

                this.return result;
            }
        }

   ##### Function pattern

        f (Some(x)) =>{ x + 1 }
        f (None()) => { 0 }

        converts to 

        f(v) =>{
            v.match().case (x:Some(x) => x + 1).case (_:None()  => 0);
        }

#### Conditions loops iterators

   ##### Conditions

        ( boolean truth).do({ 

        }).otherwise(boolean truth).do({

        }).otherwise.do({

        });

   ##### loops

        (boolean truth).loop({

        }).otherwise(boolean truth).loop({

        }).otherwise.loop({

        });

   ##### Condition and Loop mix.

        (boolean truth).do({

        }).otherwise( boolean truth).loop({

        }).otherwise(boolean truth).do({

        }).otherwise.loop({

        });

   ##### Ternary Operator

        i. s =  (boolean truth).return (some var/value).otherwise.return(some val/var);
        ii. s = (boolean truth).return (some var/val).othrewise(boolean truth).return (some var/val).otherwise.return(some var/val); 

   
#### Looping arrays/lists/maps/ranges

      arr co.lang.int -> ([5]) = [6,7,8,9,10 ];
      arr.each(idx,val).do({
             co.out.print(idx);
             co.out.print(" :: ");
             co.out.println(val);
       });

       arr co.lang.int->([5])=[22,33,41,12,98];
       arr.each(_,val).do({
             co.out.println(val);
      });

#### Array/List/Map/Range contains Element

      arr co.lang.int->([5])=[35,57,96,81,31];
      k co.lang.int = 31;
      arr.contains(k).do({
            co.out.println(k);
      }).otherwise.do({
            co.out.println("Not Found");
      });

       arr co.lang.int->([5])=[11,31,21,64,56];
       arr.contains(21).do({
             co.out.println("Found");
      }).otherwise.do({
             co.out.println("Not Found");
      });

#### Types and kinds

    x co.lang.int = 10;

    x.type() → co.lang.int x.kind() → co.lang.nothing

    x co.lang.data = 10;

    x.type() → co.lang.value x.kind() → co.lang.data so to get actual type in folang x.type().type() -> co.lang.int and it is static we can't assign x with another type at assignment type compiler will chek x.type().type() with the value's type like co.lang.int/co.lang.auto

    x co.lang.auto = 10;

    x.type() -> co.lang.int x.kind() ->co.lang.data

    //inferred at compile time and static

    x co.lang.dynammic = 10;

    x.type() -> co.lang.int x.kind()-> co.lang.data

    here x type can vary

    x (T co.lang.type)->(co.lang.type)= co.hokrt.Some(T) | co.hokrt.None();

    x.type() → co.lang.nothing x.kind() → co.lang.type->co.lang.type

#### Types

   ##### Alias:
     
        x co.lang.type = co.lang.int ;
   
   ##### New:
   
        x co.lang.newtype=co.lang.int ;

   ##### Opaque:
   
        x co.lang.opaquetype=co.lang.int ;

   ##### ADTs:

        these are tagged unions
   
        y co.lang.type=co.lang.int | co.lang.char ; 

   ##### Subtype or covariant:
   
        test co.lang.subtype = co.lang.int;

   ##### supertype or Contravariant type:
   
        test co.lang.supertype = co.lang.int;

#### Comma

    x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true;

#### grouping

    (x co.lang.int = 10, y co.lang.string = "Hello", z co.lang.bool = co.const.true);

#### Type Constructor:

        @co.dap.hokrt
        Option(T) co.lang.type = Some(T) | None();


#### Extensions

    @co.dap.extension(fortype=co.lang.string),what=extends)
    upperCase()->(string)={
        return this.upper()
    }

    @co.dap.extension(fortype=[co.lang.string]),what=overrides)
    equals(str string)->(bool)={
        this.return this == str
    }


    by declaring doesn't mean you attached and use it. It is no compiler throws error

    k co.lang.string ="abc";
    k.upperCase(); ❌ not active here

    User has to explicitly activate extensions and they are block scoped as others

    @co.dap.use(extensions=[equals,upperCase])
    now
    k.upperCase();  ✅ explicitly activated

#### Operators

    
    @co.dap.operator(symbol='+',mode=overload)
    add (a Employee, b Employee)->(Employee)={}

    @co.dap.operator(symbol='+',mode=override)
    add (a co.lang.int, b co.lang.int) = {}

  ###### The above mode override not supported in foreseable future even though valid value
  ###### Compiler throws error if mode is override. 

    @co.dap.operator(symbol='∪',mode=define,fixity=infix,associtivity=left, precedence={},arity=2)
    union (a co.lang.int, b co.lang.int)->(co.lang.int->([]))={}

    @co.dap.operator(
        symbol='∪',
        mode=define,
        fixity = infix,
        precedence = 6,
        associativity = left,
        arity = binary,
        commutative = true,
        idempotent = false,
        lazy = false,
        pure = true,
        chainable = false,
        overloadable = true,
        foldable = true,
        vectorizable = true,
        identity = 0
        distributivity=,
        associative_algebraic=true,
        desugar="intrinsic:add"
        absorption = ["∩", "∪"]  
    )


    @co.dap.operator(
    symbol='∪',
    mode=define,

    fixity=infix,
    precedence=60,
    associativity=left,
    arity=binary,
    chainable=false,

    eval=strict,
    pure=true,

    commutative=true,
    associative_algebraic=true,
    idempotent=true,
    identity="∅",

    foldable=true,
    vectorizable=false,

    distributes_over=['∩'],     // example, only if you want this
    desugar="intrinsic:set_union"
    )

    fixity

      1. infix
      2. postfix
      3. prefix
      4. circumfix
      5. postcircumfix
      6. prescircumfix
      7. mixfix
      8. ternary
      9. distfix

    
    associtivity

    precedence

    arity


#### Inline

    @co.dap.inline
    add(a co.lang.int,b co.lang.int)->(co.lang.co.lang.int) ={
        this.return a + b;
    }   

#### Lazy

    @co.dap.lazy
    x = add(1,2);

#### Dependent types

    identity( x co.lang.int) ->(x.type) = x
    
    x co.lang.dependentType->(kind=length) = co.lang.int->([5]);

######  Types computed from runtime values

    someType:= somefun(value)

    somefun(value co.lang.int)->(co.lang.type)={
        ( value < 100 ).return(co.lang.string).otherwise.return(co.lang.bool);

    }

    or
    @co.dap.typefromvalue
    somefun(value co.lang.int)->(co.lang.type)={
        ( value < 100 ).return("hello").otherwise.return(co.const.true);

    }

    Now 
    @co.dap.comptime
    @co.dap.eager
    chooseType(value co.lang.int)->(co.lang.type)={
        ( value < 100 ).return(co.lang.string).otherwise.return(co.lang.bool);

    }

    @co.dap.comptime
    somefun(value co.lang.int)->(chooseType(value co.lang.int)->(co.lang.type))={
        ( value < 100 ).return ("Hello").otherwise.return(co.const.true);

    }

    or

    somefun(value co.lang.int)->(co.lang.tag) = {
      (b < 100 ).return(co.lang.tag(co.lang.string, "Hello")).otherwise.return(co.lang.tag(co.lang.bool, co.const.true));
    }

#### Bind Variables

    $[0-9]*

#### Discard/Wildcard Variable
   _

#### Named Returns

    doManythings(a co.lang.int, b co.lang.int->(&,meta={type=out}))->(r co.lang.int, e co.lang.excpetion)={}

#### Classes and Function Chaining 

    Employee co.lang.class ={
    
        getEmployeeDetails()->(Employee) = empmodule.getEmployeeDetails; 
        //here it is assigning the module function to class's method
        
        getEmployeeInfo()->(Employee) =>> empmodule.getEmployeeDetails(); 
        //this is delegating means internally redirecting the call to module function 
    }

    //$1 $2, $3 .... are the previous results capture in $[*] bind variable to pass to next
    //methods or do something
 
     Emp co.lang.class={

     
        dosomething(a co.lang.int, b co.lang.int)->(co.lang.int)=>>somePack.somMethod(a)=>>someOthPack.somOtherMeth($1,b);
        
        //#What is $1 ?
        //#$1,$2,$3 are result from previous method
    }

#### Native functions

        @co.dap.native
        nativeMethod(a co.lang.int, b co.lang.int)->(co.lang.int) ={
            // native implementation
        }

#### Reflections
    
        @co.dap.reflection(enable=True,package="co.meta")


        x co.lang.int = 10;

        x.reflect().getType() -> co.lang.int
        x.reflect().getValue() -> 10
        x.reflect().getKind() -> value

#### Classes contd


    Employee co.lang.class ={

        @co.dap.method.static 
        getEmployee()->(Employee) ={}

        @co.dap.method.instance
        getEmployee()->(Employee)={}
        
        @co.dap.method.class  
        getEmployee()->(Employee) ={}

        @co.dap.method.object
        getEmployee()->(Employee)={}

    }


    @co.dap.oops(

        A: { inherit:true,virtual:true},// classes
        B: {implements:true},  // interfaces
        C: {inherits:true,abstract=true}, //classes
        D: {inherits:true},  // classes
        E: {uses:true},  //mixins
        F: {composes:true},  //traits
        G: {extends:true},// extension classes
        H: {with:true} //behavior or capability kind
    )
    test co.lang.class -> (uses=[],imlements=[],extends=[],inherits=[],with=package.type,composes=[]) ={

        getTest(id int)->(test) ={

        }

    }

    
#### Application Libraries

     1. Structs and Free functions

        @co.dap.library
        EmpPackage co.lang.package={
            @co.dap.export
            SEmployee co.lang.signature={
                Employee co.lang.struct;
                storeEmployee ( emp Employee)->(Employee);
    
            }

            Employee co.lang.struct={
                empId co.lang.int;
                empName co.lang.string;
            }

            storeEmployee(emp Employee)->(Employee)={
                e = Employee();
                this.return e;
            }

        }
        
    

    2. Classes 

        @co.dap.library
        EmpPackage co.lang.package={
            @co.dap.export
            IEmployee co.lang.interface={

                storeEmployee ( emp Employee)->(Employee);
        
            }

            @co.dap.oops(
                Implements:[IEmployee],
            )
            Employee co.lang.class->(implements=[IEmployee])={
                empId co.lang.int;
                empName co.lang.string;
            
                storeEmployee(emp Employee)->(Employee)={
                    e = Employee();
                    this.return e;
                }

            }

        }
    3. Modules

        @co.dap.library
        EmpPackage co.lang.package={
            @co.dap.export
            MEmployee co.lang.signature={
                Employee struct;
                storeEmployee ( emp Employee)->(Employee);
        
            }
            @co.dap.modulesig(MEmployee)           )
            impl EmployeeImpl co.lang.module->(matches=MEmployee)={
                Employee co.lang.struct = {
                    empId   co.lang.int;
                    empName co.lang.string;
                }
            
                storeEmployee(emp Employee)->(Employee)={
                    e = Employee();
                    this.return e;
                }

        }

####  Labels and Named Blocks

   ##### Labels

        outer:{
            //statements
        }      

   ##### Named Blocks

        labelBlock co.lang.block={

        }

        usage:

        labelBlock.inline();

#### Generic Types

    @co.dap.generic(typename=T)
    LinkedList co.lang.struct={
        value T
        next LinkedList
        prev LinkedList
    }
    
    k:= LinkedList.new(co.lang.int);
    
    ## here init method will be automatically invoked as structs doesn't have arg constructors

    @co.dap.generic(type={T:{typename}, R:{typename}})
    Employee co.lang.class ={

        id T
        name R
        
        @co.dap.override
        @co.dap.constructor(access=private)
        init() = {}

        co.dap.override
        @co.dap.constructor(access=public)
        init(id T, name R) = {
            this.parent.init();
            this.id = id;
            this.name = name;
        }

        getEmployee(id T)->(Employee)={}

    }

    
    a:= Employee.new(co.lang.int,co.lang.string);
    ## a is unconstructed object of type co.lang.uninit
    b: = a.init(1,"Rao");

   ###### Below example is strictly for future implementation purpose    
    
        @co.dap.generic(type={T:{typename}, R:{typename}})
        Employee co.lang.class ={

            id T
            name R
            
            // even if we don't override the new will be provided by default
            // new is very special only need not require any extended or updated information
            // override when you really want to change something with cautions

            @co.dap.method.class
            @co.dap.private
            new()->(co.lang.uninit)={ self.return co.const.none}

            @co.dap.method.class
            @co.dap.public
            new ( a co.lang.typetype, b co.lang.typetype)->(co.lang.uninit)={
                // aliasing types
                // the below is a way to handle manually rather then using @co.dap.generic 
                // @co.dap.generic will provide an automatic way to deduce types
                // without need to overrride new method
                T co.lang.type = a
                R co.lang.type = b

                // self keyword is allowed only in class methods
                self.parent.new();
                
                //uninit  instance method internally calls new and init which are private
                self.return co.lang.uninit.instance(Employee,self);
            }

            @co.dap.override
            @co.dap.constructor(access=private)
            init() = {}

            co.dap.override
            @co.dap.constructor(access=public)
            init(id T, name R) = {
                this.parent.init();
                this.id = id;
                this.name = name;
            }

            getEmployee(id T)->(Employee)={}

        }

#### Generics inheritances and types

    This is in conceptual stage not supported.

    A) Abstract vs concrete type members 
    B) Path-dependent types 
        1. Type-level projection 
        2. Path-dependent In folang how it would be


#### Comprehensions  (May be in future support)
    
    k:=(1..10).filter(|x| => x % 2 == 0).map(|x| => x * x);

    result := for (x <- List(1,2,3)).yield (x * 2)
    // List(2, 4, 6)

    // Set in → Set out
    result := for (x <- Set(1,2,3)).yield (x * 2)
    // Set(2, 4, 6)

    // Option in → Option out
    result := for (x <- Some(5)).yield (x * 2);
    // Some(10)

    // Future in → Future out
    result := for (x <- fetchData()).yield(x.process());

    //Dict in -> dict out
    ages :={"A":30,"B":40,"c":66,"e":88};
    upper := for ((name, age) <- ages).yield (name.toUpperCase, age)

#### forall
   
    forall(T) identity(x T)->(T) = {}
    forall(T) k co.lang.Maybe[T] = co.lang.Nothing;
 
    forall(T) LinkedList co.lang.struct = {
        value T;
        next LinkedList;
        prev LinkedList;
    }

    forall(T, R) Employee co.lang.class = {
        id T;
        name R;
    }

    // Constrained
    forall(T: Orderable) sort(list T->([...]))->(T->([...])) = {}

    //impredicative types ??? do we ever need for folang

#### Built in packages and methods

   ##### co

        Yes it is CO and it is one of the ReservedWord

        This is the only package provided by default

        It contains following sub packages

   ###### lang

        Contains all the data types, kinds which folang support

        a. types
        b. kinds
    
   ###### sys

        contains all the utilities to handle

        a. file
        b. concurrent 
        c. parallel
        d. goto
        e. invoke
        f. bind 
        g. call 
        h. apply 
        i. settimeout 
        j. setinterval
        k. schedular
        l. cron
        m. event

###### os

        OS Specific

        a. signal 
        b. cmd 
        c. execute  
        d. run  
        e. env 
        f. getenv
        g. setenv
        h. unsetenv
        i. sleep 
        j. exit
        k. cwd 
        l. chdir 
        m. fork
        n. wait 
        o. pipe
        p. dup
        q. close 
        r. readfd
        s. writefd
        
   ###### meta

        Meta is all about meta programming contains following

        a. patch      :  For patching exiting types, methods/functions, blocks etc
        b. instrument :  Add observability/monitoring hooks
        c. ast        :  Adding to AST mainly using macros of folang
        d. reflect    :  Reflections reading metadata and allowing modification about anything
        e. introspect :  Read only Reflection
        f. transform  :  Run structural transformations over larger graphs
        g. inject     :  Attach behavior or data from the outside
        h. create     :  Creating new things
        i. augment    :  Extend capabilities in a non-destructive way.
        j. runtime    :  Which has eval the evil function like javascript evaluates any string (must be valid folang code ) at runtime without AST changes
        
   ###### core

            Contains all the collections like

            a. list
            b. set
            c. map
            d. tree
            e. tries
            f. sort
            g. search
        
   ###### native

        Native is all about low level control for programming

        Contains Following

        a. load 
        b. register 
        c. asm       : assembly code
        d. inline    : all about machine code
        e. emit      : all about actual instruction hex codes
        f. ffi
        
   ###### in

        This Package contains all the functionalities about input

        a. read
        b. readln
        
   ###### out

        This package contaiins all the functionalities about output

        a. println
        b. print
        
   ###### regex

        Package contains all the implmentations and functions ootb for regular expression handling

        a. stex
        b. pattern
        c. match
        d. search
        
   ###### crypto

        Package contains all the implementation and functions ootb for cryptography related libs

        1. rsa
        2. aes
        3. hash
        4. md5
        5. rand
        6. uuid
        7. ssl
        8. tls

   ###### dap
   ###### ddap

        Both 11 & 12 contains built in directives, decorators, annotations and pragmas

   ###### net

        Package contains all the network related functionalities provided ootb

        1. tcp
        2. udp
        3. http
    
   ###### const

        Constants

        a. true
        b. false
        c. none
    
   ###### encoding

        Package contains the followings

        a. base64Encode
        b. base64Decode
        c. json
        d. yml
        e. bson
