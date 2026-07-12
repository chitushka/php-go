--TEST--
expect regex support
--FILE--
<?php
echo 'user-42';
--EXPECTREGEX--
user-\d+
