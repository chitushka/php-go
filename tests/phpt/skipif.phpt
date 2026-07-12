--TEST--
skipif support
--SKIPIF--
<?php
echo 'skip demonstrating SKIPIF support';
--FILE--
<?php
echo 'not executed';
--EXPECT--
not executed
