--TEST--
P2 is_object for objects aliases and closures
--FILE--
<?php
class ObjectType {}
class_alias('ObjectType', 'ObjectAlias');

$obj = new ObjectType();
$alias = new ObjectAlias();
$fn = function() {};

echo is_object($obj) ? 'object' : 'no';
echo ':';
echo is_object($alias) ? 'alias' : 'no';
echo ':';
echo is_object($fn) ? 'closure' : 'no';
echo ':';
echo is_object(array()) ? 'array' : 'no';
echo ':';
echo is_object(null) ? 'null' : 'no';
--EXPECT--
object:alias:closure:no:no
