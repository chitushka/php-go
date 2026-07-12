--TEST--
final and abstract checks
--FILE--
<?php
final class Base {}
class Child extends Base {}
--EXPECTF--
class Child cannot extend final class Base
