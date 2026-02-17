
###### arithmatic operators: +, -, *, /, %, **, ++, --
###### logical operators: &&, ||, !
###### comparison operators: ==, !=, <, >, <=, >=

###### other operators 

    λ	⒪ , â,  Ť,  ∀,  ∃,  ○,  ö, ∪, Ṡ,  Ŝ, ṁ, 𝚷, ⇛ , 𝑓 , 𝒯 , 𝘷 , 𝓕 , ↓, λ, ∂, or ⊥ ↧ or ⇓
    @, #, !, ~, $, %, ^, &, &&, (, ),_, `, <, >, /, ?, {, [, ], }, |, \, :, ;, ", ', =, ., ,

###### Reserved words

    co, let, this, impl, for

#### Variable declaration

    someVar co.lang.int;
    someString co.lang.string;

#### Variable declaration with initialization

    someBool co.lang.bool = co.const.true;
    someInt co.lang.int = 42;

#### Variable declaration with type inference

    someVal := "Hello, World!";
    someNum := 3.14;

#### pointer declaration

    somePtr co.lang.int->(*);
    someDblPtr co.lang.int->(**);

#### Array declaration

    someArray co.lang.int ->([5]);
    someDblArray co.lang.int->([2,3]);
    someJaggedArray co.lang.int->([2][3]);
    someVLArray co.lang.int->([...]);

#### Array declaration with initialization
    
    someInitializedArray co.lang.int ->([3]) = [1, 2, 3];
    someInitializedDblArray co.lang.int->([,]) = [[1, 2], [3, 4]];

#### Reference Declaration

    someRef co.lang.int ->(&);

#### LValue reference declaration

    someLValueRef co.lang.int ->(&&);

#### address declaration

    someAddr co.lang.int ->(@);

#### Thunk Declaration

    someThunk co.lang.int ->(^);

#### slice declaration
    
    someSlice co.lang.int ->([:]);

#### range declaration

    rangeI := 1..10;
    rangeS := 0<..5; 
    rangeL := 0..<100;
    rangeB := 0<..<100;
    rangeE := ..100;
    rangeF := 1..;

#### auto and dynamic variable declaration

    someAutoVar co.lang.auto;
    someDynamicVar co.lang.dynamic;

#### Fat pointers
 
    x co.lang.int->(*, kind="", meta={});


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

   ##### anonymous function
    
        @co.dap.anonymous 
        _ (k co.lang.int)->(a co.lang.int) ={
        
        }(12);


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

        closure(factor int) => (x int)=> x * factor;

        curry(factor int)(val int)= factory * val;

        curry2(x int) => (y nt) = x + y

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

        fetchEmployee(empId co.lang.string )->(Employee)=>empMod.getEmploee(this,empId);

   ##### associated functions
   
        (emp empMod.Employee) fetchEmployee(empId co.lang.string)->(empMod.Employee)=>empMod.getEmployee(emp, empId);

   ##### Generic

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

        @co.dap.indexer
        (g MyList) `[]` (index co.lang.int)->(co.lang.int) ={
            this.return g.eles[index]
        }

        @co.dap.indexer
        (g MyList) `[]=` (index co.lang.int, value co.lang.int) -> () ={
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

#### Macros

        @co.dap.macro
        yes_esc_assign()->(co.lang.untyped)={
            this.return co.macro.quote({
                co.macro.esc(y) = 42
                println("Inside macro: y = ", y)
            });
        }



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

    @co.ddap.import(path="", package="", realm="", as="")


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
    x.match(co.pattern.Type).case(co.lang.int   => ...).case(co.lang.float => ...);
    x.match(co.pattern.Value).case (0 => ...).case (1 => ...);
    x.match(co.pattern.Instance).case(xx.CAT=>...).case(xx.DOG => ...).default("Animal");
    x.match(co.pattern.Object).case(xx.Ball => "Ball").case(xx.CAT=> "CAT").default("Unknown");
        
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

   ##### Functions parameters

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

    x.type() → int x.kind() → undefined

    x co.lang.data = 10;

    x.type() → Value x.kind() → data so to get actual type in folang x.type().type() -> int and it is static we can't assign x with another type at assignment type compiler will chek x.type().type() with the value's type like co.lang.int/co.lang.auto

    x co.lang.auto = 10;

    x.type() -> int x.kind() ->data

    //inferred at compile time and static

    x co.lang.dynammic = 10;

    x.type() -> int x.kind()-> data

    here x type can vary

    x (T co.lang.type)->(co.lang.type)= co.hokrt.Some(T) | co.hokrt.None();

    x.type() → undefined x.kind() → type->type

#### Types

   ##### Alias:
     
        x co.lang.type = co.lang.int ;
   
   ##### New:
   
        x co.lang.newtype=co.lang.int ;

   ##### Opaque:
   
        x co.lang.opaquetype=co.lang.int ;

   ##### ADTs:
   
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
    Option(T) co.lang.type =  Some(T) | None();

#### for

    k co.lang.int = 1;
    for({

        k = k +1;
        co.out.println(k);
        (k >=10).do({
            this.break;
        });
    });

#### Extensions

    @co.dap.extension(fortype=co.lang.string),what=extends)
    upperCase()->(string)={
        return this.upper()
    }

    @co.dap.extension(fortype=[co.lang.string]),what=overrides)
    equals(str string)->(bool)={
        this.return this == str
    }

#### Operators

    @co.dap.operator
    `+` (a Employee, b Employee)->(Employee)={}

    @co.dap.operator(overload=true)
    `+` (a Employee, b Employee)->(Employee)={}

    @co.dap.operator(override=true)
    `+` (a co.lang.int, b co.lang.int) = {}

    @co.dap.operator(new=true)
    `∪` (a co.lang.int, b co.lang.int)->(co.lang.int->([]))={}

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

#### Bind Variables

    $[0-9]*

#### Discard/Wildcard Variable
   _

#### Named Returns

    doManythings(a co.lang.int, b co.lang.int->(&,meta={type=out}))->(r co.lang.int, e co.lang.excpetion)={}

#### Parallel:

    @co.dap.process    
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.exec
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}
        
    @co.dap.spawn    
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.fork
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}
  

#### Conurrent:
    
    @co.dap.thread 
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.task 
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}
   
    @co.dap.fiber
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

#### Async:

    @co.dap.async
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.coroutine
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.generator
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.subroutine
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.goroutine
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

#### Continuations

    @co.dap.continuation
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.continuation(type=callcc)
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.continuation(type=promptcontrol)
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.continuation(type=shiftreset)
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.continuation(type=cps)
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

#### Event

    @co.dap.event
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

#### Actors

    @co.dap.csp
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

    @co.dap.actor
    doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={}

#### Callback and defer

    @co.dap.callback
    myCallback(a co.lang.int, b co.lang.int)->()={}

    @co.dap.defer
    myDeferred(a co.lang.int, b co.lang.int)->()={}




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

     i.   co.lang.mixin
     ii.  co.lang.class
     iii. co.lang.trait
     iv.  co.lang.interface
     v.   co.lang.abstract
     vi.  co.lang.virtual
     vii. co.lang.extension
     ix. co.lang.behavior
     x. co.lang.capability

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