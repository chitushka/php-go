--TEST--
P3 strict error policy turns list assignment warning into runtime error
--INI--
php_src_go.strict_errors=1
--FILE--
<?php
list($a) = 'scalar';
echo 'bad';
?>
--EXPECT--
list assignment expects array
--EXIT_CODE--
1
