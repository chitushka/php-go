--TEST--
P3 strict error policy halts on constant and serialization warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
constant('NO_SUCH_CONST');
echo 'bad';
?>
--EXPECT--
undefined constant NO_SUCH_CONST
--EXIT_CODE--
1
