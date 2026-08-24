# Co Language Specification
### Copyright © 2025 
#### Licensed under the Creative Commons Attribution 4.0 International License.
#### https://creativecommons.org/licenses/by/4.0/
####
#### You are free to share and adapt the syntax and grammar described here,
#### provided you give appropriate credit and indicate any changes.

## Syntax Grammar (Sample)

### Variable Declaration

```plaintext
declaration := identifier ':=' expression ['as' type]
example     := count := 42 as int
```

### Function Definition

```plaintext
function := 'fun' identifier '(' [params] ')' '->' type block
params   := identifier ':' type {',' identifier ':' type}
block    := '{' statement* '}'
```

> The above syntax structure is original to the Fo Language specification.
> Please credit this source when reusing or adapting.



ō vs ô vs ö vs ọ / o̱ vs ø which one is best also rate them