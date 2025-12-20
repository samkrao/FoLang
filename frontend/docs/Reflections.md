***Reflections***

1. Package
    
    co.meta
        
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
       k. realm : kind of namespace for adding/accessing same/different class/type/struct name in different realms
