--TEST--
p0 exit construct
--FILE--
<?php
echo 'a';
exit('b');
echo 'c';
--EXPECT--
ab
