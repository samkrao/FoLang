### Generics and Parametrics

    These are valid only for Functions in colang, there is a subtle difference 
    one is runtime and the other is compile time like ( java/C# and C++)

    Curried functions, anonymous functions, lambdas cannot be generic/parametric

   1.  Compile Time () 
   
        i.
              
           a. @co.dap.generic(at=compile, 
               types={
                A: {types=[co.lang.int,co.lang.char,abc.Employee]},
                B: {types=[co.lang.float,co.lang.string]}
               }
           )
          
   2. Run time 
        
        i.
           
           a. @co.dap.generic(scope="runtime",refied=true,where="callsite")

   3. Variances

        a. Types
           
           i.   Covariant      out
           ii.  ContraVariant  in
           iii. InVariant      no tagging
   
   4. Kind

        a. Types

          i.   Param
          ii.  Return
          iii. Var
          iv.  Arg

   5. Others

       a. keys
        
          i.   Bounds
          ii.  inclusive
          iii. default
          iv.  nullable

   6. Examples

           Because we are restricting generics or parametrics to functions

           when declarring function types 

           we can specify these

          
           @co.dap.generic (
             types={
                A:{variance:contravariant, bound: sometype1,default:som,nullable:true,Kind:Param},
                B: {variance:covariant, bound: sometype2, inclusive=false,Kind:Return},
                C: {variance:invariant, bound: sometype3,default:ss,Kind: Param},
                D: {variance:covariant, bound: sometype4,nullable=false,Kind:Param},
             }
           )
           define Transformer co.lang.function= ( A, C, D) -> ( B)

           Declaring functions

           @co.dap.generic(
             at=runtime,



           )
           define   add ( a T, b T) ->(R)= { this.return a + b ;}

           @co.dap.generic(
              at=runtime,
              type={
                    T: {variance:invariant, bound=Number,Kind:Param},
                    R: {variance:invariant, bound=Number,Kind:Return}
              }   
           )
           define  add ( a T, b T) ->(R)= { this.return a + b ;}
        
            

        b. Where ?
        
           i. Use site 

           ii. Declaration site /call site

               functions mostly at declaration site



#### Constraints

     Generic functions or types must not be part of Libraries signatures/interfaces/module signatures.

    
