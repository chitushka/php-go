--TEST--
P3 ignored warning policy suppresses index assignment non-array diagnostics
--INI--
php_src_go.strict_errors=1
php_src_go.error_report=fatal,parse,error,notice
php_src_go.error_halt=fatal,parse,error,notice
--FILE--
<?php
$x = 1;
$x['key'] = 'bad';
echo $x;
echo "\n";
echo 'after';
?>
--EXPECT--
1
after
