### Novel Ideas
    
      0. Pluggable Compiler Architecture

            The compiler is designed around a **plugin-based architecture** with a strict separation between **frontend** and **backend** responsibilities.

            a. Frontend
            
                  - Frontend plugins **must be implemented in Go (Golang)**.
                  - Frontend plugins are responsible for:
                  - Parsing
                  - Syntax and Semantic analysis
                  - IR / schema generation
                  - All frontend plugins must conform to the same internal Go interfaces.
                  - **Only one frontend plugin is active per compiler instance.**

            b. Backend
            
                  - Backend plugins **may be implemented in any programming language**, provided they:
                  - Support **Protocol Buffers** for communication
                  - Adhere to the defined backend plugin protocol
                  - Backend plugins are responsible for:
                  - Code generation
                  - Target-specific transformations
                  - Producing final output artifacts

                  **Constraints:**
                  - There must be **exactly one backend plugin** per compilation run.
                  - The **default backend** is provided as a **plugin implementation written in Go**.
                  - Custom backend plugins **replace** the default backend; multiple backends are not supported.

     1. Extensible
         
      
        a. Directives/annotations

               a. Directives/prgramas
                    
                    @co.ddap.import
                    @co.ddap.builtinsshorthand

               b. Decorator/annotation
                   @co.dap.Functor 
                   Functor (F) ={

                   @co.dap.volatile
                   x co.lang.int =10;

                   @co.dap.rest(api="/emp", method=GET, pathparam=empid,format=json)
                   GetEmployee(empid string)->(Employee)=

                   @co.dap.rest(api="/emp", method=GET, queryparam=empid,format=json)
                   GetEmployees(salary co.lang.float)->([]Employee)=
                   

            

        c. Kinds as types  


            like some language where var k int;

            folang supports

            k co.lang.int

            k co.lang.struct ...

    2. Consistent

        a. Variable declaration

           c co.lang.int = 10
           d co.lang.int -> (*) = c

        b. Function declaration

           add (a co.lang.int, b co.lang.int)->(co.lang.int) = {}

        c. types

           Employee co.lang.struct = {}   


    3. Minimal

       Only 6 keywords (co,  let, impl, for,this, self)

    4. Objects and functions

       folang is a system where Objects are at the core, while the code feels functional and  
       flows fluently. That said it is neither Pure OOPS nor Pure Functinal programming language.
   
    5. Unified “postfix meta tail” syntax

        examples:

          a. Classes:

             test co.lang.class -> (uses=[],imlements=[],extends=[],inherits=[],with=package.type,composes=[]) ={

          b. Modules

             mymod co.lang.module->(matches=modulename) = {

          c. Variables

             x co.lang.int->(*, meta = { nonnull = true }) = ;
             x co.lang.int = ;

          d. Functions and/or methods

             add (x co.lang.int, y co.lang.int)->(co.lang.int) = {

          e. Others

               i.    @co.dap.Functor 
                     Functor (F) ={
               ii.   @co.dap.applicative 
                     Applicative(F) =
               iii.  @co.dap.monad 
                     Monad(F) ={
               iv.   @co.dap.monoid 
                     Monoid(T) =  
               v.    @co.dap.transformer 
                     Transformer(F(_), G(_)) = 
               vi.   @co.dap.macro 
                     debug(expr)->(co.lang.untyped)
               vii.  @co.dap.template 
                     add(a co.lang.int, b co.lang.int)->(co.lang.int) =
               viii. @co.dap.generic 
                     add ( a T, b T) ->(R)= 
               ix.   @co.dap.extension(fortype=co.lang.string),type=extends
                     upperCase()->(string)={
               x.    @co.dap.indexer 
                     [g MyList] [](index co.lang.int)->(co.lang.int) =
               xi.   packageName co.lang.package=
               xii   @co.dap.<some>
                     doSomeComplexLogic(a co.lang.int, b co.lang.int)->(co.lang.int, co.lang.Error)={
                     here some can be process, thread, async etc.
               xiii. mytype co.lang.type= co.lang.int | co.lang.float