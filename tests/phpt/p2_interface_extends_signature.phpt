--TEST--
P2 interface extends method signature compatibility
--FILE--
<?php
interface Named {
    public function name($first);
}

interface DetailedNamed extends Named {
    public function name($first, $last = '');
}

class User implements DetailedNamed {
    public function name($first, $last = '') {
        return $first . $last;
    }
}

$user = new User();
echo $user->name('Ada');
echo ':';
echo $user->name('Ada', ' Lovelace');
--EXPECT--
Ada:Ada Lovelace
