# Hello, Foλang! — Single-Source Applications

These examples use Foλang’s fixed single-source application entry file:

```text
<project>/
└── src/
    └── appl.fol
```

Each example below is a complete application. Use one example at a time as `src/appl.fol`. A single-source project has no user package directories beneath `src/`.

## 1. Hello, FoLang

The entry file is executable by definition. It does not need a `main` function or an entry-point annotation.

```folang
co.out.println("Hello, FoLang!");
```

Expected output:

```text
Hello, FoLang!
```

Project source: `examples/01-hello/src/appl.fol`

## 2. Variables and a choice

This example combines an explicitly typed binding, an inferred binding, output, and Foλang’s conditional chain.

```folang
name co.lang.string = "Kameswara";
score := 82;

co.out.print("Hello, ");
co.out.println(name);

(score >= 50).then({
    co.out.println("Result: pass");
}).default({
    co.out.println("Result: try again");
});
```

Expected output:

```text
Hello, Kameswara
Result: pass
```

`otherwise(condition)` introduces another condition. A conditionless final branch is written with `default`, as shown above.

Project source: `examples/02-variables-and-choice/src/appl.fol`

## 3. A running total

Foλang uses `loop` for repeated execution. Its receiver is the one condition tested for every iteration.

```folang
limit := 5;
current := 1;
total := 0;

(current <= limit).loop({
    total += current;
    current += 1;
});

co.out.print("Total: ");
co.out.println(total);
```

Expected output:

```text
Total: 15
```

Project source: `examples/03-running-total/src/appl.fol`

## 4. An array report

The array has a fixed length and an explicit element type. `each` receives a fresh index and value binding for every element and executes its action once.

```folang
scores co.lang.int->([5]) = [78, 92, 66, 85, 73];
total := 0;

scores.each(_, score, {
    co.out.println(score);
    total += score;
});

co.out.print("Total: ");
co.out.println(total);
```

Expected output:

```text
78
92
66
85
73
Total: 394
```

`_` discards the index binding because this program needs only each score. `each` performs the iteration itself; do not append `.loop(...)`.

Project source: `examples/04-array-report/src/appl.fol`

## 5. Hey lets find a number

I think a nuumber lets us find that number is present in a give array

```folang
number co.lang.int = 100;

arrayOfNumbers co.lang.int->([5]) = [10,20,30,40,50]
    
arrayOfNumber.contains(number).then({
    co.out.println("Hey Number " +number+ " is present !!");
}).default({
    co.ouut.println(number + " not present");
});

```

## 6. Lets use some collections

Do Folang Supports built in collections ??

Hmmm. fortunately yes, okay then lets do the find a number using list 

```folang
number co.lang.int = 100;

listOfNumbers co.core.List->(co.lang.int) = co.core.List[10,20,30,40,50];

listOfNumbes.contains(number).then({
    co.out.println("Hey Number " +number+ " is present !!");

}).default({
    co.ouut.println(number + " not present");

});

```

> Notice one thing whether it is list or built in array the way of finding an element is same in folang

## 7. Hey Lets try with a Map then

Is Map available in folang ??? 

It should be am I right otherwise how we can handle key value pairs in basic way.


```folang
number := 100;  
//hey I see colon equals and no type ? Hmmm. if you use colon equals folang infers type from intialization 

mapOfIntInt co.core.Map->(key=co.lang.int, val=co.lang.int)=co.core.Map{10:10,100:20,50:30};

maOfIntInt.contains(number).then({
    co.out.println("Hey Number " +number+ " is present !!");

}).default({
    co.ouut.println(number + " not present");

});

```
What do you think the output ?

```test
  Output: Hey Number 100 is present !!
```

So contains map means searching in key then how about searching a value is that supported ?

Great luckily yes

```folang
number := 100;  
//hey I see colon equals and no type ? Hmmm. if you use colon equals folang infers type from intialization 

mapOfIntInt co.core.Map->(key=co.lang.int, val=co.lang.int)=co.core.Map{10:10,100:20,50:30};

maOfIntInt.containsVal(number).then({
    co.out.println("Hey Number " +number+ " is present !!");

}).default({
    co.ouut.println(number + " not present");

});

```

## 8. Hey What about stacks and queues

Sure.

```folang
number := 100;  


stack co.core.Stack->(co.lang.int)=co.core.Stack[10:10,100:20,50:30];

stack.contains(number).then({
    co.out.println("Hey Number " +number+ " is present !!");

}).default({
    co.ouut.println(number + " not present");

});

```

```folang
number := 100;  


queue co.core.Queue->(co.lang.int)=co.core.Queue[10:10,100:20,50:30];

Queue.contains(number).then({
    co.out.println("Hey Number " +number+ " is present !!");

}).default({
    co.ouut.println(number + " not present");

});

```

I am seeing Stack, Queue, List, and built in Arrays looking same 

You are right they are all same but they expose different set of methods for working

Stack has pop, push,peek 
Queue has enque,deque, add, remove front 
List  next and previous, add and remove
Array indexed access 

They look same use for different purposes


## 9. Hey do we have sets 

Ofcourse folang has built in set but it is slightly different from list, queues


```folang
number := 100;  


set co.core.Set->(co.lang.int)=co.core.Set(10:10,100:20,50:30);

set.contains(number).then({
    co.out.println("Hey Number " +number+ " is present !!");

}).default({
    co.ouut.println(number + " not present");

});

```
Ha ! everywhere contains ?  yes folang provides uniform way to check element presence

## 10. I want to work with ranges 

You chosen right programming language 


```folang
// Typed range variable declaration

someRange co.lang.int->(..);

// Inferred range declarations

rangeI := 1 .. 10; // [1, 10] ExcludeStart=false, ExcludeEnd=false

rangeS := 0 <.. 5; // (0, 5] ExcludeStart=true, ExcludeEnd=false

rangeL := 0 ..< 100; // [0, 100) ExcludeStart=false, ExcludeEnd=true

rangeB := 0 <..< 100; // (0, 100) ExcludeStart=true, ExcludeEnd=true

rangeE := .. 100; // open lower bound (_, 100]

rangeF := 1 ..; // open upper bound [1, _)
```

You can use same each and contains methods on range to iterate or find values.


## 11. How can I use external libraries you are mentioning

Get the library downloaded place it in project-root/lib folder compile by importing the Library

To import library 
    1. packaged export libraries
        @co.ddap.import(package="somepackage.someusb", alias="sp" );
    2. Pure library
        @co.ddap.import(library="someExternal",alias="se")  // where alias must be uniquue
        

and use with aliases like `co.out.println`

## 12. Why co.* has no imports then

Great question co.* is always available that doesn't mean you can use every package in it. Please refere langauage-reference manual for more information.


## Are there a way to create user defined data types

Great point. This book is all about showing how to start coding in folang

If you enjoyed this journey and would like to learn how Foλang approaches and structures a larger application step by step, please have a look at "Think Like Foλang".

Where we show some of other features and why folang architects using them for a specified problem.