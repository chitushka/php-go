--TEST--
interface constants
--FILE--
<?php
interface Named {
    const KIND = 'named';
}

interface Detailed extends Named {
    const VERSION = 53;
}

class User implements Detailed {
}

echo Named::KIND;
echo "\n";
echo Detailed::KIND;
echo ':';
echo Detailed::VERSION;
echo "\n";
echo User::KIND;
echo ':';
echo User::VERSION;
--EXPECT--
named
named:53
named:53
