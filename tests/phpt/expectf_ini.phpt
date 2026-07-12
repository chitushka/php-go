--TEST--
expectf and ini sections
--INI--
display_errors=1
--FILE--
<?php
echo 'value ';
echo 42;
--EXPECTF--
value %d
