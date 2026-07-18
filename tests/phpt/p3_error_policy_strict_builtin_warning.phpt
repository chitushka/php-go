--TEST--
P3 strict error policy turns registry builtin warning into runtime error
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
strlen();
echo 'bad';
?>
--EXPECT--
strlen expects 1 arguments, got 0
--EXIT_CODE--
1
