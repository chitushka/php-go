--TEST--
P3 SAPI builtin argument warnings are non-halting by default
--FILE--
<?php
$a = header();
$b = call_user_func('headers_list', 'x');
echo ($a === null ? 'null' : 'bad');
echo "\n";
echo ($b === null ? 'null' : 'bad');
echo "\n";
echo 'after';
?>
--EXPECT--
null
null
after
