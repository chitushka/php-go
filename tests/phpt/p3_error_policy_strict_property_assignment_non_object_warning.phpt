--TEST--
P3 strict error policy halts on property assignment non-object warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
$x = 1;
$x->name = 'bad';
echo 'bad';
?>
--EXPECT--
cannot assign property name on non-object
--EXIT_CODE--
1
