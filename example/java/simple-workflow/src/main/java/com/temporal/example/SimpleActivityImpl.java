package com.temporal.example;

/**
 * Implementation of the SimpleActivity interface.
 * This activity provides a simple greeting functionality.
 */
public class SimpleActivityImpl implements SimpleActivity {
    
    @Override
    public String greet(String name) {
        return "Hello, " + name + "!";
    }
}
