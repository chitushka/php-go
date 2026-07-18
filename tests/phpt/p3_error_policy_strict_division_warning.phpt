--TEST--
P3 strict error policy turns division warning into runtime error
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
$value = 5 / 0;
echo 'bad';
?>
--EXPECT--
Division by zero
--EXIT_CODE--
1
