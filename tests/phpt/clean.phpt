--TEST--
clean section support
--FILE--
<?php
echo 'clean-ok';
--CLEAN--
<?php
$cleanup = 1;
--EXPECT--
clean-ok
