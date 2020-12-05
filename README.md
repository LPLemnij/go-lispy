To use this, simply clone the project and type go build and run the executable.
All of the builtin 'out of the box' functionality is documented inside of builtin.go and lenv.go. From there, feel free to do whatever you want.

Potential future plans:

1. Add a macro system 
2. Add golang interop with the golang standard library as this would provide huge levels of functionality
3. Define a standard library for this lisp that just wraps around the golang stl.
4. Add better error handling and prettier printing.

This was an excercise based on the book found at buildyourownlisp.com. The book originally has it in C so I translated that and added/improved on some things.
