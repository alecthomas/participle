# Benchmarks

Parser             | Lexer              | Speed (MiB/s)
-------------------|--------------------|---------------
Generated          | text/scanner       | 28.5
Reflection         | text/scanner       | 6.6
Generated          | Stateful           | 3.59
Reflection         | Stateful           | 2.43
Generated          | Generated          | 15.3
Reflection         | Generated          | 5.25

## Conclusion

It appears the generated parser code is limited by the throughput of the
stateful lexer. Putting more effort into improving the generated stateful
lexer code would likely provide significant performance improvements.