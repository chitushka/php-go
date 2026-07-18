--TEST--
P3 ignored notice policy suppresses scalar read access diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,warning
php_src_go.error_halt=fatal,parse,error,warning
--FILE--
<?php
$x = 1;
$value = $x->name;
echo ($value === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
after
