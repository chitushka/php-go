--TEST--
P2 reserved class names are rejected
--FILE--
<?php
class self {}
?>
--EXPECTF--
parse errors: [cannot use 'self' as class name as it is reserved]
