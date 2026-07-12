--TEST--
interface implementation error
--FILE--
<?php
interface Named {
    public function name();
}

class User implements Named {}
--EXPECTF--
class User must implement interface method Named::name
