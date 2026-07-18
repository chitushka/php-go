--TEST--
P3 error policy can ignore list assignment warnings and continue
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
list($a) = 'scalar';
echo isset($a) ? 'set' : 'null';
echo "\n";
echo 'after';
?>
--EXPECT--
null
after
