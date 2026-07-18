--TEST--
P3 strict error policy halts on preg_match capture warnings
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
preg_match('/x/', 'x', 'not-writable');
echo 'bad';
?>
--EXPECT--
preg_match matches argument must be writable
--EXIT_CODE--
1
