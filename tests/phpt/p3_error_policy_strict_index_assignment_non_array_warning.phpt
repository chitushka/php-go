--TEST--
P3 strict error policy halts on index assignment non-array warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
$x = 1;
$x['key'] = 'bad';
echo 'bad';
?>
--EXPECT--
cannot assign index on non-array
--EXIT_CODE--
1
