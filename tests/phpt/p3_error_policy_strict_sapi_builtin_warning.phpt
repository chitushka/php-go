--TEST--
P3 strict error policy turns SAPI builtin warning into runtime error
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
header();
echo 'bad';
?>
--EXPECT--
header expects between 1 and 2 arguments, got 0
--EXIT_CODE--
1
