--TEST--
P3 strict error policy halts on user function missing argument warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
function needs($value) {}
needs();
echo 'bad';
?>
--EXPECT--
function needs expects at least 1 arguments, got 0
--EXIT_CODE--
1
