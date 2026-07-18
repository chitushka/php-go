--TEST--
P3 strict error policy halts on OOP introspection warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
get_class(1);
echo 'bad';
?>
--EXPECT--
get_class expects object
--EXIT_CODE--
1
