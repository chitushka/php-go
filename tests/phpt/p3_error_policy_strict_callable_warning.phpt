--TEST--
P3 strict error policy halts on callable builtin warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
call_user_func();
echo 'bad';
?>
--EXPECT--
call_user_func expects at least 1 argument, got 0
--EXIT_CODE--
1
